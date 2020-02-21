package globals

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strconv"

	"github.com/dave/dst"
)

// RenamePkg for rename package
func RenamePkg(df *dst.File, pkgName string) {
	df.Name.Name = pkgName
}

// GetImportMap retrieve import map from ast.File
func GetImportMap(f *ast.File) (m map[string]string /* import name -> path*/) {
	m = make(map[string]string)

	// prefer file.Decls to file.Imports
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.IMPORT {
			continue
		}

		for _, gs := range d.Specs {
			s := gs.(*ast.ImportSpec)
			path, err := strconv.Unquote(s.Path.Value)
			if err != nil {
				panic(fmt.Sprintf("strconv.Unquote:%v", err))
			}
			if s.Name != nil {
				m[s.Name.Name] = path
			} else {
				m[filepath.Base(path)] = path
			}
		}
	}
	return
}

// WalkGlobalsDst will walk over all global identifiers
func WalkGlobalsDst(df *dst.File, f func(name string, kind SymKind) bool) (err error) {
	for _, d := range df.Decls {
		switch td := d.(type) {
		case *dst.GenDecl:
			switch td.Tok {
			case token.IMPORT:
				for _, s := range td.Specs {
					ts := s.(*dst.ImportSpec)
					if ts.Name != nil {
						if !f(ts.Name.Name, KindImport) {
							return
						}
					} else {
						var path string
						path, err = strconv.Unquote(ts.Path.Value)
						if err != nil {
							return
						}
						if !f(filepath.Base(path), KindImport) {
							return
						}
					}

				}
			case token.TYPE:
				for _, s := range td.Specs {
					ts := s.(*dst.TypeSpec)
					if !f(ts.Name.Name, KindType) {
						return
					}
				}
			case token.CONST, token.VAR:
				kind := KindConst
				if td.Tok == token.VAR {
					kind = KindVar
				}
				for _, s := range td.Specs {
					ts := s.(*dst.ValueSpec)
					for _, nameIdent := range ts.Names {
						if !f(nameIdent.Name, kind) {
							return
						}
					}
				}
			}
		case *dst.FuncDecl:
			if td.Recv == nil && !f(td.Name.Name, KindFunc) {
				return
			}
		}
	}

	return
}

// AddImports for add imports to file
func AddImports(df *dst.File, imports map[string]string) {
	specs := make([]dst.Spec, 0, len(imports))
	for name, path := range imports {
		if name == filepath.Base(path) {
			name = ""
		}
		specs = append(specs, &dst.ImportSpec{
			Name: &dst.Ident{Name: name},
			Path: &dst.BasicLit{Value: strconv.Quote(path)},
		})
	}

	d := &dst.GenDecl{
		Tok:    token.IMPORT,
		Specs:  specs,
		Lparen: true,
	}

	newDecls := make([]dst.Decl, 0, len(df.Decls)+1)
	newDecls = append(newDecls, d)
	newDecls = append(newDecls, df.Decls...)
	df.Decls = newDecls
}

// UpdateConstValue for update global constant value
func UpdateConstValue(df *dst.File, consts map[string]string) {
	for _, decl := range df.Decls {
		d, ok := decl.(*dst.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}

		for _, gs := range d.Specs {
			s := gs.(*dst.ValueSpec)
			for i, id := range s.Names {
				if n, ok := consts[id.Name]; ok {
					s.Values[i] = &dst.BasicLit{Value: n}
				}
			}
		}
	}
}

// RemoveDecl for remove global declares
func RemoveDecl(df *dst.File, names []string) {
	if len(names) == 0 {
		return
	}
	nmap := make(map[string]struct{})
	for _, name := range names {
		nmap[name] = struct{}{}
	}
	dIdx2Del := make(map[int]struct{})
	for di, d := range df.Decls {
		switch td := d.(type) {
		case *dst.GenDecl:
			switch td.Tok {
			case token.IMPORT:
				// TODO update file.Imports if needed
				sIdx2Del := make(map[int]struct{})
				for si, s := range td.Specs {
					s := s.(*dst.ImportSpec)
					var name string
					if s.Name == nil {
						str, err := strconv.Unquote(s.Path.Value)
						if err != nil {
							panic(fmt.Sprintf("strconv.Unquote:%v", err))
						}
						name = filepath.Base(str)
					} else if s.Name.Name != "_" {
						name = s.Name.Name
					}
					if _, exists := nmap[name]; !exists {
						continue
					}

					if len(td.Specs) > 1 {
						sIdx2Del[si] = struct{}{}
					} else {
						dIdx2Del[di] = struct{}{}
					}
				}
				if len(sIdx2Del) > 0 {
					var specs []dst.Spec
					for si, s := range td.Specs {
						if _, exists := sIdx2Del[si]; !exists {
							specs = append(specs, s)
						}
					}
					td.Specs = specs
				}

			case token.TYPE:
				sIdx2Del := make(map[int]struct{})
				for si, s := range td.Specs {
					s := s.(*dst.TypeSpec)
					name := s.Name.Name

					if _, exists := nmap[name]; !exists {
						continue
					}

					if len(td.Specs) > 1 {
						sIdx2Del[si] = struct{}{}
					} else {
						dIdx2Del[di] = struct{}{}
					}
				}
				if len(sIdx2Del) > 0 {
					var specs []dst.Spec
					for si, s := range td.Specs {
						if _, exists := sIdx2Del[si]; !exists {
							specs = append(specs, s)
						}
					}
					td.Specs = specs
				}
			case token.CONST, token.VAR:
				sIdx2Del := make(map[int]struct{})
				for si, s := range td.Specs {
					s := s.(*dst.ValueSpec)
					nIdx2Del := make(map[int]struct{})
					for ni, ident := range s.Names {
						name := ident.Name

						if _, exists := nmap[name]; !exists {
							continue
						}

						if len(s.Names) > 1 {
							nIdx2Del[ni] = struct{}{}
						} else if len(td.Specs) > 1 {
							sIdx2Del[si] = struct{}{}
						} else {
							dIdx2Del[di] = struct{}{}
						}
					}
					// TODO fix comment
					if len(nIdx2Del) > 0 {
						var (
							idents []*dst.Ident
							values []dst.Expr
						)
						for ni, ident := range s.Names {
							if _, exists := nIdx2Del[ni]; !exists {
								idents = append(idents, ident)
							}
						}
						for vi, expr := range s.Values {
							if _, exists := nIdx2Del[vi]; !exists {
								values = append(values, expr)
							}
						}
						s.Names = idents
						s.Values = values
					}
				}
				if len(sIdx2Del) > 0 {
					var specs []dst.Spec
					for si, s := range td.Specs {
						if _, exists := sIdx2Del[si]; !exists {
							specs = append(specs, s)
						}
					}
					td.Specs = specs
				}
			}
		case *dst.FuncDecl:
			if td.Recv != nil {
				continue
			}
			name := td.Name.Name
			if _, exists := nmap[name]; exists {
				dIdx2Del[di] = struct{}{}
			}
		}
	}
	if len(dIdx2Del) > 0 {
		var decls []dst.Decl
		for di, d := range df.Decls {
			if _, exists := dIdx2Del[di]; !exists {
				decls = append(decls, d)
			}
		}
		df.Decls = decls
	}
}

// UpdateComment for update comment of global declares
func UpdateComment(df *dst.File, cf func(name string, node dst.Node)) {
	for _, d := range df.Decls {
		switch td := d.(type) {
		case *dst.GenDecl:
			switch td.Tok {
			case token.TYPE:
				for _, s := range td.Specs {
					s := s.(*dst.TypeSpec)
					name := s.Name.Name

					if len(td.Specs) > 1 {
						cf(name, s)
					} else {
						cf(name, td)
					}

				}
			case token.CONST, token.VAR:
				for _, s := range td.Specs {
					s := s.(*dst.ValueSpec)
					for _, ident := range s.Names {
						name := ident.Name

						if len(td.Specs) > 1 {
							cf(name, s)
						} else {
							cf(name, td)
						}
					}

				}
			}
		case *dst.FuncDecl:
			if td.Recv != nil {
				continue
			}
			name := td.Name.Name
			cf(name, td)
		}
	}
}

// GetIdent returns the identifier associated with the given expression by
// removing parentheses if needed.
func GetIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.ParenExpr:
		return GetIdent(e.X)
	default:
		return nil
	}
}

// GetIdentDst for dst
func GetIdentDst(expr dst.Expr) *dst.Ident {
	switch e := expr.(type) {
	case *dst.Ident:
		return e
	case *dst.ParenExpr:
		return GetIdentDst(e.X)
	default:
		return nil
	}
}
