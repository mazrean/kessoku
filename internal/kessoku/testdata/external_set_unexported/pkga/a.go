package pkga

import "github.com/mazrean/kessoku"

// A is provided by Set.
type A struct{}

func newA() *A {
	return &A{}
}

// Set exposes an unexported provider function.
var Set = kessoku.Set(kessoku.Provide(newA))
