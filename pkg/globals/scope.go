package globals

// SymKind specifies the kind of a global symbol. For example, a variable, const
// function, etc.
type SymKind int

// Constants for different kinds of symbols.
const (
	KindUnknown SymKind = iota
	KindFunc
	KindImport
	KindType
	KindConst
	KindVar
	KindReceiver
	KindParameter
	KindResult
)

type symbol struct {
	kind  SymKind
	scope *scope
}

type scope struct {
	outer *scope
	syms  map[string]*symbol
}

func newScope(outer *scope) *scope {
	return &scope{
		outer: outer,
		syms:  make(map[string]*symbol),
	}
}

func (s *scope) isGlobal() bool {
	return s.outer == nil
}

func (s *scope) lookup(n string) *symbol {
	return s.syms[n]
}

func (s *scope) deepLookup(n string) *symbol {
	for x := s; x != nil; x = x.outer {
		if sym := x.lookup(n); sym != nil {
			return sym
		}
	}
	return nil
}

func (s *scope) add(name string, kind SymKind) {
	if s.syms[name] != nil {
		return
	}

	s.syms[name] = &symbol{
		kind:  kind,
		scope: s,
	}
}
