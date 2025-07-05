package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*Service](
	"InitializeComplexService",
	kessoku.Provide(NewConfig),
	kessoku.Value("example value"),
	kessoku.As[Interface](kessoku.Provide(NewConcreteImpl)),
	kessoku.Provide(NewService),
)
