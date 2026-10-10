package pkgb

import "github.com/mazrean/kessoku"

// B is provided by Set.
type B struct{}

// NewB creates a new B.
func NewB() *B {
	return &B{}
}

// Set groups the providers of this package.
var Set = kessoku.Set(kessoku.Provide(NewB))
