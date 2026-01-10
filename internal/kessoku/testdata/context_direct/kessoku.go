package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test providers that take context.Context directly (without Async)
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewService),
)
