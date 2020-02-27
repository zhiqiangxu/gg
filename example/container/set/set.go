// Package set provides the implementation of set.
// run: gg -i set.go -t Type=string -d Set=StringSet -d NewSet=NewStringSet -d NewKeySet=NewKeyStringSet
package set

type (
	// Type will be erased after template instantiation
	Type  interface{}
	empty struct{}
)

// Set of Type elements
type Set map[Type]empty

// NewSet creates a Set from a list of values.
func NewSet(items ...Type) Set {
	s := Set{}
	s.Insert(items...)
	return s
}

// NewKeySet creates a Set from a keys of a map
func NewKeySet(theMap map[Type]empty) Set {
	s := Set{}

	for k := range theMap {
		s.Insert(k)
	}
	return s
}

// Insert adds items to the set.
func (s Set) Insert(items ...Type) Set {
	for _, item := range items {
		s[item] = empty{}
	}
	return s
}

// Delete removes all items from the set.
func (s Set) Delete(items ...Type) Set {
	for _, item := range items {
		delete(s, item)
	}
	return s
}

// Has returns true if and only if item is contained in the set.
func (s Set) Has(item string) bool {
	_, contained := s[item]
	return contained
}

// Len returns the size of the set.
func (s Set) Len() int {
	return len(s)
}
