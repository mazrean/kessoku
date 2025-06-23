package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*Service](
	"InitializeComplexService",
	kessoku.Provide(NewConfig),
	kessoku.Value("example value"),
	kessoku.Bind[Interface](kessoku.Provide(NewConcreteImpl)),
	kessoku.Arg[int]("arg"),
	kessoku.Provide(NewService),
)
