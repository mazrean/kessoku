package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test that error-only validator providers are not silently dropped
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewDB),
	kessoku.Provide(ValidateDB),
	kessoku.Provide(NewService),
)
