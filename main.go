package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"go.uber.org/zap"

	"github.com/zhiqiangxu/gg/pkg/globals"
	"github.com/zhiqiangxu/gg/pkg/merge"
	"github.com/zhiqiangxu/util/logger"
)

var (
	output      = flag.String("o", "", "output `file`")
	debug       = flag.Bool("debug", false, "`debug` mode")
	suffix      = flag.String("suffix", "", "`suffix` to add to each global symbol")
	prefix      = flag.String("prefix", "", "`prefix` to add to each global symbol")
	packageName = flag.String("p", "", "output package `name`")
	inFiles     []string
	types       = make(map[string]string)
	declares    = make(map[string]string)
	consts      = make(map[string]string)
	imports     = make(map[string]string)
)

type sliceValue []string

func (sv *sliceValue) String() string {
	var b bytes.Buffer
	first := true
	for _, v := range *sv {
		if !first {
			b.WriteRune(',')
		} else {
			first = false
		}
		b.WriteString(v)
	}
	return b.String()
}

func (sv *sliceValue) Set(s string) error {
	*sv = append(*sv, s)

	return nil
}

// mapValue implements flag.Value. We use a mapValue flag instead of a regular
// string flag when we want to allow more than one instance of the flag. For
// example, we allow several "-d A=B" arguments, and will rename them all.
type mapValue map[string]string

func (m mapValue) String() string {
	var b bytes.Buffer
	first := true
	for k, v := range m {
		if !first {
			b.WriteRune(',')
		} else {
			first = false
		}
		b.WriteString(k)
		b.WriteRune('=')
		b.WriteString(v)
	}
	return b.String()
}

func (m mapValue) Set(s string) error {
	sep := strings.Index(s, "=")
	if sep == -1 {
		return fmt.Errorf("missing '=' from '%s'", s)
	}

	m[s[:sep]] = s[sep+1:]

	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Var((*sliceValue)(&inFiles), "i", "specify the input file. Multiple files are allowed by multiple -i.")
	flag.Var(mapValue(declares), "d", "rename global A(can be either of Type/Var/Func/Const) to B when `A=B` is passed in. Multiple such mappings are allowed.")
	flag.Var(mapValue(consts), "c", "reassign constant A to value B when `A=B` is passed in. Multiple such mappings are allowed.")
	flag.Var(mapValue(types), "t", "replace type A to type B when `A=B` is passed in. Multiple such mappings are allowed.")
	flag.Var(mapValue(imports), "import", "add new imports. `name=path` specifies that 'name', used in types as name.type, refers to the package living in 'path'.")
	flag.Parse()

	// *input = "test/data/walk_test_data.go"
	// *output = "test2.go"
	// declares = map[string]string{"GlobalType": "GlobalType2"}

	if len(inFiles) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var (
		mergedCode string
		err        error
	)
	if len(inFiles) > 1 {
		mergedCode, err = merge.PackageFiles(inFiles)
		if err != nil {
			logger.Instance().Fatal("PackageFiles", zap.Error(err))
		}
	}

	// Parse the input file.
	fset := token.NewFileSet()
	var f *ast.File
	if mergedCode != "" {
		f, err = parser.ParseFile(fset, "", mergedCode, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
	} else {
		f, err = parser.ParseFile(fset, inFiles[0], nil, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
	}
	if err != nil {
		logger.Instance().Fatal("ParseFile", zap.Error(err))
	}

	// check params
	checkParams(f)

	// types are treated similar to declares, except that the old type will be removed at lasat
	if declares == nil {
		declares = make(map[string]string)
	}
	for k, v := range types {
		declares[k] = v
	}

	if *packageName != "" {
		globals.RenamePkg(f, *packageName)
	}
	globals.UpdateConstValue(f, consts)
	// used for changing comment
	new2old := map[string]string{}
	globals.RenameDecl(fset, f, func(ident *ast.Ident, kind globals.SymKind) {
		old := ident.Name
		if declares[ident.Name] != "" {
			ident.Name = declares[ident.Name]
		}
		ident.Name = *prefix + ident.Name + *prefix
		new2old[ident.Name] = old
	})

	{
		// ast -> dst for comment
		df, err := decorator.DecorateFile(fset, f)
		if err != nil {
			logger.Instance().Fatal("ecorator.DecorateFile", zap.Error(err))
		}

		if *debug {
			fmt.Println("new2old", new2old)
		}

		// update comments
		{
			globals.UpdateComment(df, func(newName string, node dst.Node) {
				oldName := new2old[newName]
				if newName == oldName {
					return
				}

				for i, comment := range node.Decorations().Start {
					node.Decorations().Start[i] = strings.ReplaceAll(comment, oldName, newName)
				}
			})
		}

		// remove replaced types
		{
			var types2Remove []string
			for _, name := range types {
				types2Remove = append(types2Remove, name)
			}
			globals.RemoveDecl(df, types2Remove)
		}

		// add imports
		if len(imports) > 0 {
			globals.AddImports(df, imports)
		}

		// dst -> ast
		fset, f, err = decorator.RestoreFile(df)
		if err != nil {
			logger.Instance().Fatal("ecorator.RestoreFile", zap.Error(err))
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		logger.Instance().Fatal("format.Node", zap.Error(err))
	}

	if *output == "" {
		fmt.Println(buf.String())
		return
	}

	if err := ioutil.WriteFile(*output, buf.Bytes(), 0644); err != nil {
		logger.Instance().Fatal("WriteFile", zap.Error(err))
	}
}

func checkParams(f *ast.File) {
	importMap := globals.GetImportMap(f)

	for _, exprStr := range types {
		expr, err := parser.ParseExpr(exprStr)
		if err != nil {
			logger.Instance().Fatal("parser.ParseExpr", zap.Error(err))
		}
		ast.Inspect(expr, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				id := globals.GetIdent(x.X)
				if id == nil {
					logger.Instance().Fatal("invalid SelectorExpr", zap.Any("SelectorExpr", x))
				}
				importName := id.Name
				if importMap[importName] == "" && imports[importName] == "" {
					logger.Instance().Fatal("invalid importName", zap.String("importName", importName), zap.Any("SelectorExpr", x))
				}
			}
			return true
		})
	}

}
