package globals

import (
	"fmt"
	"go/token"
	"path/filepath"
	"strconv"

	"github.com/dave/dst"
)

type walker struct {
	// file is the file whose nodes are being visited.
	df *dst.File

	// f is the visit function to be called when a global symbol is reached.
	f func(*dst.Ident, SymKind)

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

func (w *walker) walkExpr(e dst.Expr) {
	switch te := e.(type) {
	case nil:
	case *dst.Ident:
		if s := w.scope.deepLookup(te.Name); s != nil && s.scope.isGlobal() {
			w.f(te, s.kind)
		}
	case *dst.Ellipsis:
		w.walkExpr(te.Elt)
	case *dst.BasicLit:
	case *dst.FuncLit:
		w.pushScope()
		w.walkFieldList(te.Type.Params, KindParameter)
		w.walkFieldList(te.Type.Results, KindResult)
		w.walkBlockStmt(te.Body)
		w.popScope()
	case *dst.CompositeLit:
		w.walkExpr(te.Type)
		for _, elt := range te.Elts {
			w.walkExpr(elt)
		}
	case *dst.ParenExpr:
		w.walkExpr(te.X)
	case *dst.SelectorExpr:
		w.walkExpr(te.X)
	case *dst.IndexExpr:
		w.walkExpr(te.X)
		w.walkExpr(te.Index)
	case *dst.SliceExpr:
		w.walkExpr(te.X)
		w.walkExpr(te.Low)
		w.walkExpr(te.High)
		w.walkExpr(te.Max)
	case *dst.TypeAssertExpr:
		w.walkExpr(te.X)
		if te.Type != nil {
			w.walkExpr(te.Type)
		}
	case *dst.CallExpr:
		w.walkCallExpr(te)
	case *dst.StarExpr:
		w.walkExpr(te.X)
	case *dst.UnaryExpr:
		w.walkExpr(te.X)
	case *dst.BinaryExpr:
		w.walkExpr(te.X)
		w.walkExpr(te.Y)
	case *dst.KeyValueExpr:
		w.walkExpr(te.Value)
	case *dst.ArrayType:
		w.walkExpr(te.Len)
		w.walkExpr(te.Elt)
	case *dst.StructType:
		w.walkFieldList(te.Fields, KindUnknown)
	case *dst.FuncType:
		w.walkFieldList(te.Params, KindUnknown)
		w.walkFieldList(te.Results, KindUnknown)
	case *dst.InterfaceType:
		w.walkFieldList(te.Methods, KindUnknown)
	case *dst.MapType:
		w.walkExpr(te.Key)
		w.walkExpr(te.Value)
	case *dst.ChanType:
		w.walkExpr(te.Value)
	}
}

func (w *walker) walkFieldList(l *dst.FieldList, kind SymKind) {
	if l == nil {
		return
	}

	for _, f := range l.List {
		for _, n := range f.Names {
			if kind != KindUnknown {
				w.scope.add(n.Name, kind)
			}
		}
		w.walkExpr(f.Type)
	}
}

func (w *walker) walkCallExpr(ce *dst.CallExpr) {
	w.walkExpr(ce.Fun)

	for i := 0; i < len(ce.Args); i++ {
		w.walkExpr(ce.Args[i])
	}
}

func (w *walker) walkStmt(s dst.Stmt) {
	switch ts := s.(type) {
	case nil, *dst.BranchStmt, *dst.EmptyStmt:
	case *dst.DeclStmt:
		w.walkDecl(ts.Decl, false)
	case *dst.LabeledStmt:
		w.walkStmt(ts.Stmt)
	case *dst.ExprStmt:
		w.walkExpr(ts.X)
	case *dst.SendStmt:
		w.walkExpr(ts.Chan)
		w.walkExpr(ts.Value)
	case *dst.IncDecStmt:
		w.walkExpr(ts.X)
	case *dst.AssignStmt:
		for _, e := range ts.Rhs {
			w.walkExpr(e)
		}

		for _, e := range ts.Lhs {
			if ts.Tok == token.DEFINE {
				if n := GetIdentDst(e); n != nil {
					w.scope.add(n.Name, KindVar)
				}
			}
			w.walkExpr(e)
		}
	case *dst.GoStmt:
		w.walkCallExpr(ts.Call)
	case *dst.DeferStmt:
		w.walkCallExpr(ts.Call)
	case *dst.ReturnStmt:
		for _, e := range ts.Results {
			w.walkExpr(e)
		}
	case *dst.BlockStmt:
		w.walkBlockStmt(ts)
	case *dst.IfStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkExpr(ts.Cond)
		w.walkBlockStmt(ts.Body)
		w.walkStmt(ts.Else)
		w.popScope()
	case *dst.SwitchStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkExpr(ts.Tag)
		for _, bs := range ts.Body.List {
			c := bs.(*dst.CaseClause)
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
	case *dst.TypeSwitchStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkStmt(ts.Assign)
		for _, cs := range ts.Body.List {
			c := cs.(*dst.CaseClause)
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
	case *dst.SelectStmt:
		for _, cs := range ts.Body.List {
			c := cs.(*dst.CommClause)

			w.pushScope()
			w.walkStmt(c.Comm)
			for _, bs := range c.Body {
				w.walkStmt(bs)
			}
			w.popScope()
		}
	case *dst.ForStmt:
		w.pushScope()
		w.walkStmt(ts.Init)
		w.walkExpr(ts.Cond)
		w.walkStmt(ts.Post)
		w.walkBlockStmt(ts.Body)
		w.popScope()
	case *dst.RangeStmt:
		w.pushScope()
		w.walkExpr(ts.X)
		if ts.Tok == token.DEFINE {
			if n := GetIdentDst(ts.Key); n != nil {
				w.scope.add(n.Name, KindVar)
			}

			if n := GetIdentDst(ts.Value); n != nil {
				w.scope.add(n.Name, KindVar)
			}
		}
		w.walkExpr(ts.Key)
		w.walkExpr(ts.Value)
		w.walkBlockStmt(ts.Body)
		w.popScope()
	default:
		w.unexpected()
	}
}

func (w *walker) unexpected() {
	panic(fmt.Sprintf("Unable to parse at"))
}

func (w *walker) walkBlockStmt(bs *dst.BlockStmt) {
	w.pushScope()

	for _, s := range bs.List {
		w.walkStmt(s)
	}

	w.popScope()
}

func (w *walker) walkDecl(d dst.Decl, phase1 bool) {
	switch td := d.(type) {
	case *dst.GenDecl:
		switch td.Tok {
		case token.IMPORT:

			for _, s := range td.Specs {
				s := s.(*dst.ImportSpec)
				var name string
				if s.Name == nil {
					str, err := strconv.Unquote(s.Path.Value)
					if err != nil {
						panic(fmt.Sprintf("strconv.Unquote:%v", err))
					}
					name = filepath.Base(str)
					w.scope.add(name, KindImport)
				} else if s.Name.Name != "_" {
					name = s.Name.Name
					w.scope.add(name, KindImport)
				}
				if !phase1 && name != "" {
					ident := dst.NewIdent(name)
					w.f(ident, KindImport)
					if ident.Name != name {
						s.Name = ident
					}
				}

			}
		case token.TYPE:
			for _, s := range td.Specs {
				s := s.(*dst.TypeSpec)

				w.scope.add(s.Name.Name, KindType)
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
				s := s.(*dst.ValueSpec)
				if !phase1 {
					if s.Type != nil {
						w.walkExpr(s.Type)
					}

					for _, e := range s.Values {
						w.walkExpr(e)
					}
				}

				for _, n := range s.Names {
					w.scope.add(n.Name, kind)
					if !phase1 {
						if w.scope.isGlobal() {
							w.f(n, kind)
						}
					}
				}
			}
		}
	case *dst.FuncDecl:
		if td.Recv == nil {
			w.scope.add(td.Name.Name, KindFunc)
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
	for _, d := range w.df.Decls {
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
func RenameDecl(df *dst.File, f func(*dst.Ident, SymKind)) {
	v := walker{
		df: df,
		f:  f,
	}

	v.walk()
}
