package test

import (
	"go/parser"
	"go/token"
	"testing"

	"reflect"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/zhiqiangxu/gg/pkg/globals"
	"github.com/zhiqiangxu/gg/pkg/merge"
)

func TestGlobals(t *testing.T) {
	// Parse the input file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "data/walk_test_data.go", nil, parser.ParseComments|parser.DeclarationErrors|parser.SpuriousErrors)
	if err != nil {
		t.Fatal("ParseFile", err)
	}

	df, err := decorator.DecorateFile(fset, f)
	if err != nil {
		t.Fatal("DecorateFile", err)
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
	globals.RenameDecl(df, func(ident *dst.Ident, kind globals.SymKind) {
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

	output, err := merge.PackageFiles(inFiles)
	if err != nil {
		t.Fatal("PackageFiles", err, output)
	}

	// ioutil.WriteFile("merged.go", []byte(output), 0644)

}
