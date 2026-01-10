package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test sync providers with error returns (not async)
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewService),
)
