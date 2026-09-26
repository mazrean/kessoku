package setpkg

import (
	"github.com/mazrean/kessoku"
	dep "github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_import_alias/depimpl"
)

// Wrapper wraps D.
type Wrapper struct {
	D *dep.D
}

// NewWrapper creates a new Wrapper.
func NewWrapper(d *dep.D) *Wrapper {
	return &Wrapper{D: d}
}

// Set references depimpl through the local alias dep.
var Set = kessoku.Set(kessoku.Provide(dep.NewD), kessoku.Provide(NewWrapper))
