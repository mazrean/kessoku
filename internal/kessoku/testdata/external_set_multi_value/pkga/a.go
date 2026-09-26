package pkga

import "github.com/mazrean/kessoku"

// A is provided by Set.
type A struct{}

// NewA creates a new A.
func NewA() *A { return &A{} }

func pair[T any](a, b T) (T, T) { return a, b }

// Unused and Set are initialized by a single multi-value call.
var Unused, Set = pair(kessoku.Set(), kessoku.Set(kessoku.Provide(NewA)))
