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
	"log"
	"os"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/zhiqiangxu/go2gen/pkg/globals"
)

var (
	input       = flag.String("i", "", "input `file`")
	output      = flag.String("o", "", "output `file`")
	debug       = flag.Bool("debug", false, "`debug` mode")
	suffix      = flag.String("suffix", "", "`suffix` to add to each global symbol")
	prefix      = flag.String("prefix", "", "`prefix` to add to each global symbol")
	packageName = flag.String("p", "", "output package `name`")
	declares    = make(map[string]string)
	consts      = make(map[string]string)
)

// mapValue implements flag.Value. We use a mapValue flag instead of a regular
// string flag when we want to allow more than one instance of the flag. For
// example, we allow several "-t A=B" arguments, and will rename them all.
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

	flag.Var(mapValue(declares), "d", "rename global A(can be either of Type/Var/Func/Const) to B when `A=B` is passed in. Multiple such mappings are allowed.")
	flag.Var(mapValue(consts), "c", "reassign constant A to value B when `A=B` is passed in. Multiple such mappings are allowed.")
	flag.Parse()

	// *input = "test/data/walk_test_data.go"
	// *output = "test2.go"
	// declares = map[string]string{"GlobalType": "GlobalType2"}

	if *input == "" || *output == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Parse the input file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *input, nil, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
	if err != nil {
		log.Fatal("ParseFile", err)
	}

	if *packageName != "" {
		globals.RenamePkg(f, *packageName)
	}
	globals.ModifyConst(f, consts)
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
		df, err := decorator.DecorateFile(fset, f)
		if err != nil {
			log.Fatal("ecorator.DecorateFile", err)
		}

		if *debug {
			fmt.Println("new2old", new2old)
		}

		for _, d := range df.Decls {
			switch td := d.(type) {
			case *dst.GenDecl:
				switch td.Tok {
				case token.TYPE:
					for _, s := range td.Specs {
						s := s.(*dst.TypeSpec)
						newName := s.Name.Name
						oldName := new2old[newName]
						if newName == oldName {
							continue
						}
						for i, comment := range s.Decorations().Start {
							s.Decorations().Start[i] = strings.ReplaceAll(comment, oldName, newName)
						}
					}
				case token.CONST, token.VAR:
					for _, s := range td.Specs {
						s := s.(*dst.ValueSpec)
						for _, ident := range s.Names {
							newName := ident.Name
							oldName := new2old[newName]
							if newName == oldName {
								continue
							}

							for i, comment := range s.Decorations().Start {
								s.Decorations().Start[i] = strings.ReplaceAll(comment, oldName, newName)
							}
						}

					}
				}
			case *dst.FuncDecl:
				newName := td.Name.Name
				oldName := new2old[newName]
				if newName == oldName {
					continue
				}
				for i, comment := range td.Decorations().Start {
					td.Decorations().Start[i] = strings.ReplaceAll(comment, oldName, newName)
				}
			}
		}
		fset, f, err = decorator.RestoreFile(df)
		if err != nil {
			log.Fatal("ecorator.RestoreFile", err)
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		log.Fatal("format.Node", err)
	}

	if err := ioutil.WriteFile(*output, buf.Bytes(), 0644); err != nil {
		log.Fatal("WriteFile", err)
	}
}
