package pkgc

import "github.com/mazrean/kessoku"

// C is shared.
type C struct{}

// NewC creates a new C.
func NewC() *C {
	return &C{}
}

// Set groups the providers of this package.
var Set = kessoku.Set(kessoku.Provide(NewC))
