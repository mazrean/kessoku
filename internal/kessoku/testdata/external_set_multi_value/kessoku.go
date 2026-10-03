//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_multi_value/pkga"
)

var _ = kessoku.Inject[*pkga.A](
	"InitializeA",
	pkga.Set,
)

func main() {}
