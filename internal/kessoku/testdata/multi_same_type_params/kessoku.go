package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test that a provider with multiple parameters of the same type
// generates separate injector parameters for each position.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewService),
)
