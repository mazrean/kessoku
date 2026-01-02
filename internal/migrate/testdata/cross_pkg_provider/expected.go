//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/migrate/testdata/cross_pkg_provider/pkg"
)

var ProviderSet = kessoku.Set(
	kessoku.Provide(pkg.NewFoo),
	kessoku.Provide(pkg.NewBar),
)
var _ = kessoku.Inject[*pkg.Foo](
	"InitializeApp",
	ProviderSet,
)
