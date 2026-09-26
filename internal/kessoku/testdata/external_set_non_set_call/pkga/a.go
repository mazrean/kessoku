package pkga

import "github.com/mazrean/kessoku"

// A is provided by Set.
type A struct{}

// NewA creates a new A.
func NewA() *A { return &A{} }

func identity[T any](v T) T { return v }

// Set is built through a helper instead of directly by kessoku.Set.
var Set = identity(kessoku.Set(kessoku.Provide(NewA)))
