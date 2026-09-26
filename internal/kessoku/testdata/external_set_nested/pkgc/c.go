package pkgc

import "github.com/mazrean/kessoku"

// C holds configuration.
type C struct {
	Name string
}

// NewC creates a new C.
func NewC() *C {
	return &C{Name: "c"}
}

// Set groups the providers of this package.
var Set = kessoku.Set(kessoku.Provide(NewC))
