package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test that a provider returning a concrete error type (*MyError) generates
// correct nil-check code that does not trigger a false positive when the
// provider returns a logically-nil error.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewService),
)
