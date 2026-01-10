package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
