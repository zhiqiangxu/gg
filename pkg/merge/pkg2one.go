package merge

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	"github.com/zhiqiangxu/util/logger"
	"go.uber.org/zap"
	"golang.org/x/tools/imports"
)

var (
	// ErrPkgNameInconsistent when package name inconsistent
	ErrPkgNameInconsistent = errors.New("package name inconsistent")
)

// Pkg2One for merge multiple package files into one
func Pkg2One(inFiles []string) (output string, err error) {
	files := make(map[string]*ast.File)
	fset := token.NewFileSet()
	var name string
	for _, fname := range inFiles {
		var f *ast.File
		f, err = parser.ParseFile(fset, fname, nil, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
		if err != nil {
			logger.Instance().Error("ParseFile", zap.Error(err))
			return
		}

		files[fname] = f
		if name == "" {
			name = f.Name.Name
		} else if name != f.Name.Name {
			logger.Instance().Error("package name inconsistent", zap.String("name1", name), zap.String("name2", f.Name.Name))
			err = ErrPkgNameInconsistent
			return
		}
	}

	// Merge all files into one.
	pkg := &ast.Package{
		Name:  name,
		Files: files,
	}
	f := ast.MergePackageFiles(pkg, ast.FilterUnassociatedComments|ast.FilterFuncDuplicates|ast.FilterImportDuplicates)

	// Write the output file.
	var buf bytes.Buffer
	if err = format.Node(&buf, fset, f); err != nil {
		logger.Instance().Error("format.Node", zap.Error(err))
		return
	}

	outputBytes, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		logger.Instance().Error("imports.Process", zap.Error(err))
		return
	}

	output = string(outputBytes)
	return
}
