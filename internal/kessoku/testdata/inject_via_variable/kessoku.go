package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

var injectFn = kessoku.Inject[*Service]

var _ = injectFn(
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
