package pkga

import "github.com/mazrean/kessoku"

// A is provided by Set.
type A struct{ N int }

// Set uses the len builtin inside a closure.
var Set = kessoku.Set(kessoku.Provide(func() *A { return &A{N: len("abc")} }))
