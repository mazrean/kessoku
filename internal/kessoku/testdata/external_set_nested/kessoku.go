//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_nested/pkga"
)

// InitializeLabel wires a Set that nests Sets the target file never imports.
var _ = kessoku.Inject[pkga.Label](
	"InitializeLabel",
	pkga.Set,
)

func main() {}
