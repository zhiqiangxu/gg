package test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"reflect"

	"github.com/zhiqiangxu/yag/pkg/globals"
	"github.com/zhiqiangxu/yag/pkg/merge"
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
		"root":        globals.KindType,
		"main":        globals.KindFunc,
		"GlobalFunc":  globals.KindFunc,
		"GlobalVars":  globals.KindVar,
		"GlobalConst": globals.KindConst,
	}
	got := make(map[string]globals.SymKind)
	globals.RenameDecl(fset, f, func(ident *ast.Ident, kind globals.SymKind) {
		if kind == globals.KindImport {
			return
		}
		got[ident.Name] = kind
	})

	if !reflect.DeepEqual(expect, got) {
		t.Fatal("expect != got")
	}
}

func TestMerge(t *testing.T) {

	inFiles := []string{"data/merge/f1.go", "data/merge/f2.go"}

	outSrc, err := merge.Pkg2One(inFiles)
	if err != nil {
		t.Fatal("Pkg2One", err)
	}

	fmt.Println(outSrc)
}
