//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_builtin_shadow/pkga"
)

var _ = kessoku.Inject[*pkga.A](
	"InitializeA",
	pkga.Set,
)

func len(string) int { return 0 }

func main() {}
