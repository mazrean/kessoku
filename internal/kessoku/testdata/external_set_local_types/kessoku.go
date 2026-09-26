//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_local_types/pkga"
)

// InitializeResult uses a Set with local and anonymous types.
var _ = kessoku.Inject[*pkga.Result](
	"InitializeResult",
	pkga.Set,
)

func main() {}
