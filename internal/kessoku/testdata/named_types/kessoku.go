package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test custom named types as dependencies
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewAPIKey),
	kessoku.Provide(NewDatabaseURL),
	kessoku.Provide(NewService),
)
