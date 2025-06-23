package main

//go:generate go tool kessoku $GOFILE

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/examples/cross_package/providers"
)

var _ = kessoku.Inject[*providers.ExternalService](
	"InitializeCrossPackageService",
	kessoku.Provide(providers.NewExternalConfig),
	kessoku.Provide(providers.NewExternalService),
)
