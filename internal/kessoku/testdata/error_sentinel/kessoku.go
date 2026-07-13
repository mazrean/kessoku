package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test that kessoku.Value((error)(nil)) forces error return even when no provider returns error.
// This is emitted by `kessoku migrate` to preserve a wire injector's (*T, error) signature.
var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Value((error)(nil)),
	kessoku.Provide(NewApp),
)
