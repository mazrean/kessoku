//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/mazrean/kessoku/internal/migrate/testdata/cross_pkg_provider/pkg"
)

var ProviderSet = wire.NewSet(
	pkg.NewFoo,
	pkg.NewBar,
)

func InitializeApp() *pkg.Foo {
	wire.Build(ProviderSet)
	return nil
}
