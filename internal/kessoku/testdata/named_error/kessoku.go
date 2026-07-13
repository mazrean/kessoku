package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test provider returning a named concrete error type (not the error interface).
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
