package test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"reflect"

	"github.com/zhiqiangxu/go2gen/pkg/globals"
)

func TestGlobals(t *testing.T) {
	// Parse the input file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "data/walk_test_data.go", nil, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
	if err != nil {
		t.Fatal("ParseFile", err)
	}

	expect := map[string]globals.SymKind{
		"GlobalType":  globals.KindType,
		"GlobalFunc":  globals.KindFunc,
		"GlobalVars":  globals.KindVar,
		"GlobalConst": globals.KindConst,
	}
	got := make(map[string]globals.SymKind)
	globals.RenameDecl(fset, f, func(ident *ast.Ident, kind globals.SymKind) {
		got[ident.Name] = kind
	})

	reflect.DeepEqual(expect, got)
}
