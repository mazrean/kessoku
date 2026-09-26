//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_shadowing/pkga"
)

// InitializeLabel uses a Set whose closures shadow package names.
var _ = kessoku.Inject[pkga.Label](
	"InitializeLabel",
	pkga.Set,
)

func main() {}
