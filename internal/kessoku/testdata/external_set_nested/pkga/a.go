package pkga

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_nested/pkgc"
)

// A depends on pkgc.C.
type A struct {
	Name string
}

// Prefix is an exported package-level value.
var Prefix = "a:"

// NewPrefixed creates a new A from C.
func NewPrefixed(c *pkgc.C) *A {
	return &A{Name: Prefix + c.Name}
}

// Label is provided by a function literal.
type Label string

// internalSet is unexported, but only exposes exported identifiers.
var internalSet = kessoku.Set(
	kessoku.Provide(NewPrefixed),
)

// Set nests an unexported Set and a Set from another package.
var Set = kessoku.Set(
	internalSet,
	pkgc.Set,
	kessoku.Provide(func(a *A) Label {
		l := A{Name: Prefix + a.Name}
		return Label(l.Name)
	}),
)
