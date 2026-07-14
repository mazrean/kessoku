package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Async(kessoku.Provide(NewConfig)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Provide(NewService),
)
