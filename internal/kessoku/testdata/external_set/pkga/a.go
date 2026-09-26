package pkga

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set/pkgb"
)

// A depends on pkgb.B.
type A struct {
	b *pkgb.B
}

// NewA creates a new A.
func NewA(b *pkgb.B) *A {
	return &A{b: b}
}

// Set groups the providers of this package.
var Set = kessoku.Set(kessoku.Provide(NewA))
