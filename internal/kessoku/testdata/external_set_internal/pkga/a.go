package pkga

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_internal/pkga/internal/impl"
)

// A is provided by Set.
type A struct{ N int }

// Set uses an internal package of pkga.
var Set = kessoku.Set(kessoku.Provide(func() *A { return &A{N: impl.NewValue()} }))
