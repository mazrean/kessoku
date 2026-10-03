package pkga

import "github.com/mazrean/kessoku"

// A has an unexported field.
type A struct{ n int }

// Set builds A with an unkeyed literal.
var Set = kessoku.Set(kessoku.Provide(func() *A { return &A{1} }))
