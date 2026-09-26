package pkga

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_duplicate/pkgc"
)

// A depends on C.
type A struct {
	C *pkgc.C
}

// NewA creates a new A.
func NewA(c *pkgc.C) *A {
	return &A{C: c}
}

// Set also includes pkgc.Set.
var Set = kessoku.Set(pkgc.Set, kessoku.Provide(NewA))
