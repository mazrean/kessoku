//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_internal/pkga"
)

var _ = kessoku.Inject[*pkga.A](
	"InitializeA",
	pkga.Set,
)

func main() {}
