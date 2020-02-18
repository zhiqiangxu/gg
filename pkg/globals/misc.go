package globals

import (
	"go/ast"
	"go/token"
)

// RenamePkg for rename package
func RenamePkg(file *ast.File, pkgName string) {
	file.Name.Name = pkgName
}

// ModifyConst for modify global constants
func ModifyConst(file *ast.File, consts map[string]string) {
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
