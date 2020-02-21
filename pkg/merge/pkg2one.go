package merge

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"

	"github.com/zhiqiangxu/gg/pkg/globals"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/zhiqiangxu/util/logger"
	"go.uber.org/zap"
)

var (
	// ErrPkgNameInconsistent when package name inconsistent
	ErrPkgNameInconsistent = errors.New("package name inconsistent")
)

// PackageFiles merges multiple files belong to the same package into one
func PackageFiles(inFiles []string) (output string, err error) {
	if len(inFiles) == 0 {
		return
	}

	// collect all files as dst.File
	files := make([]*dst.File, 0, len(inFiles))
	fset := token.NewFileSet()
	var name string
	for _, fname := range inFiles {
		var (
			f  *ast.File
			df *dst.File
		)
		f, err = parser.ParseFile(fset, fname, nil, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
		if err != nil {
			logger.Instance().Error("ParseFile", zap.Error(err))
			return
		}

		df, err = decorator.DecorateFile(fset, f)
		if err != nil {
			logger.Instance().Error("DecorateFile", zap.Error(err))
			return
		}
		files = append(files, df)
		if name == "" {
			name = df.Name.Name
		} else if name != df.Name.Name {
			logger.Instance().Error("package name inconsistent", zap.String("name1", name), zap.String("name2", f.Name.Name))
			err = ErrPkgNameInconsistent
			return
		}
	}

	// start merge process with dst.File
	type nameAndPath struct {
		name string
		path string
	}
	var (
		sortedImports []*dst.ImportSpec
		nap2dfs       = make(map[nameAndPath][]*dst.File)
		nameDupNaps   = make(map[nameAndPath]bool)
		importMap     = make(map[nameAndPath]*dst.ImportSpec)
	)

	{
		nameSeen := make(map[string]bool)
		// first find all import declares
		for _, df := range files {
			// TODO maybe also support import the same package multiple times with different names
			for _, d := range df.Decls {
				if td, ok := d.(*dst.GenDecl); ok && td.Tok == token.IMPORT {
					for _, s := range td.Specs {
						s := s.(*dst.ImportSpec)

						var path string
						path, err = strconv.Unquote(s.Path.Value)
						if err != nil {
							logger.Instance().Error("strconv.Unquote", zap.Error(err))
							return
						}

						var importName string
						if s.Name != nil {
							importName = s.Name.Name
						} else {
							importName = filepath.Base(path)
						}

						key := nameAndPath{name: importName, path: path}

						if !nameSeen[importName] {
							nameSeen[importName] = true
						} else {
							if !(importMap[key] != nil && importMap[key].Path.Value == s.Path.Value) {
								nameDupNaps[key] = true
							}
						}

						nap2dfs[key] = append(nap2dfs[key], df)
						if importMap[key] != nil {
							continue
						}

						importMap[key] = s
						sortedImports = append(sortedImports, s)
					}
				}
			}
		}
	}

	// check if it's possible to reuse the original import name
	nameCheckFunc := func(candidateName string) bool {
		pass := true
		for _, df := range files {
			globals.WalkGlobalsDst(df, func(name string, kind globals.SymKind) bool {
				if kind != globals.KindImport && candidateName == name {
					pass = false
					return false
				}
				return true
			})
			if !pass {
				break
			}
		}
		return pass
	}
	// find import name to change
	toChange := make(map[nameAndPath]string)
	for nap, s := range importMap {
		pass := !nameDupNaps[nap] && nameCheckFunc(nap.name)
		if !pass {
			// can not reuse, have to generate a new import name
			baseName := filepath.Base(nap.path)
			idx := 0
			nameGenerator := func() (name string) {
				if idx >= 100 {
					return ""
				}
				name = fmt.Sprintf("%s%02d", baseName, idx)
				idx++
				return
			}

			var finalName string
			// try each candidate name
			for {
				candidateName := nameGenerator()
				if candidateName == "" {
					err = fmt.Errorf("candidateName used up for %s", baseName)
					return
				}

				if nameCheckFunc(candidateName) {
					finalName = candidateName
					break
				}
			}
			toChange[nap] = finalName
			// change s will take effect in sortedImports
			s.Name = dst.NewIdent(finalName)
		}
	}

	// rename all files with the decided name
	for nap, importName := range toChange {
		for _, df := range nap2dfs[nap] {
			globals.RenameDecl(df, func(ident *dst.Ident, kind globals.SymKind) {
				if kind == globals.KindImport && ident.Name == nap.name {
					ident.Name = importName
				}
			})
		}
	}

	// clear for reuse
	importMap = make(map[nameAndPath]*dst.ImportSpec)

	// collect non-import declares after rename
	var nonimportDecls []dst.Decl
	for _, df := range files {
		for _, d := range df.Decls {
			if td, ok := d.(*dst.GenDecl); ok && td.Tok == token.IMPORT {
			} else {
				nonimportDecls = append(nonimportDecls, d)
			}
		}
	}

	// prepend sortedImports as a single declare to the decls
	importDecl := &dst.GenDecl{Tok: token.IMPORT, Specs: make([]dst.Spec, len(sortedImports))}
	if len(sortedImports) > 1 {
		importDecl.Lparen = true
	}
	for i := 0; i < len(sortedImports); i++ {
		importDecl.Specs[i] = sortedImports[i]
	}

	decls := make([]dst.Decl, 0, len(nonimportDecls)+1)
	decls = append(decls, importDecl)
	decls = append(decls, nonimportDecls...)

	mdf := &dst.File{Name: files[0].Name, Decs: files[0].Decs, Decls: decls}

	// dst -> ast
	var mf *ast.File
	fset, mf, err = decorator.RestoreFile(mdf)
	if err != nil {
		logger.Instance().Error("RestoreFile", zap.Error(err))
	}

	// Write the output file.
	var buf bytes.Buffer
	if err = format.Node(&buf, fset, mf); err != nil {
		logger.Instance().Error("format.Node", zap.Error(err))
		return
	}

	output = buf.String()
	return
}
