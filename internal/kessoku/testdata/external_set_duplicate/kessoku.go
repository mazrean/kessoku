//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_duplicate/pkga"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_duplicate/pkgc"
)

// InitializeA reaches pkgc.Set both directly and through pkga.Set.
var _ = kessoku.Inject[*pkga.A](
	"InitializeA",
	pkga.Set,
	pkgc.Set,
)

func main() {}
