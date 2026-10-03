//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set/pkga"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set/pkgb"
)

// InitializeA wires Sets declared in other packages.
var _ = kessoku.Inject[*pkga.A](
	"InitializeA",
	pkga.Set,
	pkgb.Set,
)

// InitializeApp reuses the same Sets in a second injector.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	pkga.Set,
	pkgb.Set,
	kessoku.Provide(NewApp),
)
