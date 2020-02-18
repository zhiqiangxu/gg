package globals

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strconv"
)

type walker struct {
	// file is the file whose nodes are being visited.
	file *ast.File

	// fset is the file set the file being visited belongs to.
	fset *token.FileSet

	// f is the visit function to be called when a global symbol is reached.
	f func(*ast.Ident, SymKind)

	// scope is the current scope as nodes are visited.
	scope *scope
}

// pushScope creates a new scope and pushes it to the top of the scope stack.
func (w *walker) pushScope() {
	w.scope = newScope(w.scope)
}

// popScope removes the scope created by the last call to pushScope.
func (w *walker) popScope() {
	w.scope = w.scope.outer
}

func (w *walker) walkExpr(e ast.Expr) {
	switch te := e.(type) {
	case nil:
	case *ast.Ident:
		if s := w.scope.deepLookup(te.Name); s != nil && s.scope.isGlobal() {
			w.f(te, s.kind)
		}
	case *ast.Ellipsis:
		w.walkExpr(te.Elt)
	case *ast.BasicLit:
	case *ast.FuncLit:
		w.pushScope()
		w.walkFieldList(te.Type.Params, KindParameter)
		w.walkFieldList(te.Type.Results, KindResult)
		w.walkBlockStmt(te.Body)
		w.popScope()
	case *ast.CompositeLit:
		w.walkExpr(te.Type)
		for _, elt := range te.Elts {
			w.walkExpr(elt)
		}
	case *ast.ParenExpr:
		w.walkExpr(te.X)
	case *ast.SelectorExpr:
		w.walkExpr(te.X)
	case *ast.IndexExpr:
		w.walkExpr(te.X)
		w.walkExpr(te.Index)
	case *ast.SliceExpr:
		w.walkExpr(te.X)
		w.walkExpr(te.Low)
		w.walkExpr(te.High)
		w.walkExpr(te.Max)
	case *ast.TypeAssertExpr:
		w.walkExpr(te.X)
		if te.Type != nil {
			w.walkExpr(te.Type)
		}
	case *ast.CallExpr:
		w.walkCallExpr(te)
	case *ast.StarExpr:
		w.walkExpr(te.X)
	case *ast.UnaryExpr:
		w.walkExpr(te.X)
	case *ast.BinaryExpr:
		w.walkExpr(te.X)
		w.walkExpr(te.Y)
	case *ast.KeyValueExpr:
		w.walkExpr(te.Value)
	case *ast.ArrayType:
		w.walkExpr(te.Len)
		w.walkExpr(te.Elt)
	case *ast.StructType:
		w.walkFieldList(te.Fields, KindUnknown)
	case *ast.FuncType:
		w.walkFieldList(te.Params, KindUnknown)
		w.walkFieldList(te.Results, KindUnknown)
	case *ast.InterfaceType:
		w.walkFieldList(te.Methods, KindUnknown)
	case *ast.MapType:
		w.walkExpr(te.Key)
		w.walkExpr(te.Value)
	case *ast.ChanType:
		w.walkExpr(te.Value)
	}
}

func (w *walker) walkFieldList(l *ast.FieldList, kind SymKind) {
	if l == nil {
		return
	}

	for _, f := range l.List {
		for _, n := range f.Names {
			if kind != KindUnknown {
				w.scope.add(n.Name, kind, n.Pos())
			}
		}
		w.walkExpr(f.Type)
	}
}

func (w *walker) walkCallExpr(ce *ast.CallExpr) {
	w.walkExpr(ce.Fun)

	for i := 0; i < len(ce.Args); i++ {
		w.walkExpr(ce.Args[i])
	}
}

func (w *walker) walkStmt(s ast.Stmt) {
	switch ts := s.(type) {
	case nil, *ast.BranchStmt, *ast.EmptyStmt:
	case *ast.DeclStmt:
		w.walkDecl(ts.Decl, false)
	case *ast.LabeledStmt:
		w.walkStmt(ts.Stmt)
	case *ast.ExprStmt:
		w.walkExpr(ts.X)
	case *ast.SendStmt:
		w.walkExpr(ts.Chan)
		w.walkExpr(ts.Value)
	case *ast.IncDecStmt:
		w.walkExpr(ts.X)
	case *ast.AssignStmt:
		for _, e := range ts.Rhs {
			w.walkExpr(e)
		}

		for _, e := range ts.Lhs {
			if ts.Tok == token.DEFINE {
				if n := GetIdent(e); n != nil {
					w.scope.add(n.Name, KindVar, n.Pos())
				}
			}
			w.walkExpr(e)
		}
	case *ast.GoStmt:
		w.walkCallExpr(ts.Call)
	case *ast.DeferStmt:
		w.walkCallExpr(ts.Call)
	case *ast.ReturnStmt:
		for _, e := range ts.Results {
			w.walkExpr(e)
		}
	case *ast.BlockStmt:
		w.walkBlockStmt(ts)
	case *ast.IfStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkExpr(ts.Cond)
		w.walkBlockStmt(ts.Body)
		w.walkStmt(ts.Else)
		w.popScope()
	case *ast.SwitchStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkExpr(ts.Tag)
		for _, bs := range ts.Body.List {
			c := bs.(*ast.CaseClause)
			w.pushScope()
			for _, ce := range c.List {
				w.walkExpr(ce)
			}
			for _, bs := range c.Body {
				w.walkStmt(bs)
			}
			w.popScope()
		}
		w.popScope()
	case *ast.TypeSwitchStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkStmt(ts.Assign)
		for _, cs := range ts.Body.List {
			c := cs.(*ast.CaseClause)
			w.pushScope()
			for _, ce := range c.List {
				w.walkExpr(ce)
			}
			for _, bs := range c.Body {
				w.walkStmt(bs)
			}
			w.popScope()
		}
		w.popScope()
	case *ast.SelectStmt:
		for _, cs := range ts.Body.List {
			c := cs.(*ast.CommClause)

			w.pushScope()
			w.walkStmt(c.Comm)
			for _, bs := range c.Body {
				w.walkStmt(bs)
			}
			w.popScope()
		}
	case *ast.ForStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkExpr(ts.Cond)
		w.walkStmt(ts.Post)
		w.walkBlockStmt(ts.Body)
		w.popScope()
	case *ast.RangeStmt:
		w.pushScope()
		w.walkExpr(ts.X)
		if ts.Tok == token.DEFINE {
			if n := GetIdent(ts.Key); n != nil {
				w.scope.add(n.Name, KindVar, n.Pos())
			}

			if n := GetIdent(ts.Value); n != nil {
				w.scope.add(n.Name, KindVar, n.Pos())
			}
		}
		w.walkExpr(ts.Key)
		w.walkExpr(ts.Value)
		w.walkBlockStmt(ts.Body)
		w.popScope()
	default:
		w.unexpected(s.Pos())
	}
}

func (w *walker) unexpected(p token.Pos) {
	panic(fmt.Sprintf("Unable to parse at %v", w.fset.Position(p)))
}

func (w *walker) walkBlockStmt(bs *ast.BlockStmt) {
	w.pushScope()

	for _, s := range bs.List {
		w.walkStmt(s)
	}

	w.popScope()
}

func (w *walker) walkDecl(d ast.Decl, phase1 bool) {
	switch td := d.(type) {
	case *ast.GenDecl:
		switch td.Tok {
		case token.IMPORT:

			for _, s := range td.Specs {
				s := s.(*ast.ImportSpec)
				var name string
				if s.Name == nil {
					str, err := strconv.Unquote(s.Path.Value)
					if err != nil {
						panic(fmt.Sprintf("strconv.Unquote:%v", err))
					}
					name = filepath.Base(str)
					w.scope.add(name, KindImport, s.Path.Pos())
				} else if s.Name.Name != "_" {
					name = s.Name.Name
					w.scope.add(name, KindImport, s.Name.Pos())
				}
				if !phase1 && name != "" {
					ident := ast.NewIdent(name)
					w.f(ident, KindImport)
					if ident.Name != name {
						s.Name = ident
					}
				}

			}
		case token.TYPE:
			for _, s := range td.Specs {
				s := s.(*ast.TypeSpec)

				w.scope.add(s.Name.Name, KindType, s.Name.Pos())
				if phase1 {
					return
				}

				if w.scope.isGlobal() {
					w.f(s.Name, KindType)
				}

				w.walkExpr(s.Type)

			}
		case token.CONST, token.VAR:
			kind := KindConst
			if td.Tok == token.VAR {
				kind = KindVar
			}

			for _, s := range td.Specs {
				s := s.(*ast.ValueSpec)
				if !phase1 {
					if s.Type != nil {
						w.walkExpr(s.Type)
					}

					for _, e := range s.Values {
						w.walkExpr(e)
					}
				}

				for _, n := range s.Names {
					w.scope.add(n.Name, kind, n.Pos())
					if !phase1 {
						if w.scope.isGlobal() {
							w.f(n, kind)
						}
					}
				}
			}
		}
	case *ast.FuncDecl:
		if td.Recv == nil {
			w.scope.add(td.Name.Name, KindFunc, td.Name.Pos())
		}

		if !phase1 {
			if td.Recv == nil {
				if w.scope.isGlobal() {
					w.f(td.Name, KindFunc)
				}
			}

			w.pushScope()
			w.walkFieldList(td.Recv, KindReceiver)
			w.walkFieldList(td.Type.Params, KindParameter)
			w.walkFieldList(td.Type.Results, KindResult)
			if td.Body != nil {
				w.walkBlockStmt(td.Body)
			}
			w.popScope()
		}
	}
}

// 一共有Decl、Spec、Stmt、Expr和Node这5个interface
// File包含[]Decl，Decl分为GenDecl和FuncDecl
// GenDecl的Spec分为ImportSpec、TypeSpec和ValueSpec
// ValueSpec包含var和const，FuncDecl包含BlockStmt，BlockStmt包含[]Stmt
// Stmt包含Expr，一切都是Node
func (w *walker) walkFile(phase1 bool) {
	for _, d := range w.file.Decls {
		w.walkDecl(d, phase1)
	}
}

func (w *walker) walk() {
	w.pushScope()

	w.walkFile(true)

	w.walkFile(false)

}

// RenameDecl traverses the provided AST and calls f() for each identifier that
// refers to global declares. The global declare must be defined in the file itself.
//
// The function f() is allowed to modify the identifier, for example, to rename
// uses of global references.
func RenameDecl(fset *token.FileSet, file *ast.File, f func(*ast.Ident, SymKind)) {
	v := walker{
		fset: fset,
		file: file,
		f:    f,
	}

	v.walk()
}
