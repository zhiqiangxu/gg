package globals

import (
	"go/ast"
	"go/token"

	"github.com/dave/dst"
)

// RenamePkg for rename package
func RenamePkg(file *ast.File, pkgName string) {
	file.Name.Name = pkgName
}

// UpdateConstValue for update global constant value
func UpdateConstValue(file *ast.File, consts map[string]string) {
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}

		for _, gs := range d.Specs {
			s := gs.(*ast.ValueSpec)
			for i, id := range s.Names {
				if n, ok := consts[id.Name]; ok {
					s.Values[i] = &ast.BasicLit{Value: n}
				}
			}
		}
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
