package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test that a provider returning a named concrete error type (not the error
// interface itself) is not mis-identified as the error return slot.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
