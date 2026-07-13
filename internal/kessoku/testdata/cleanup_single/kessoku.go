//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Test that a single provider returning (*T, func()) defers the cleanup func.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewApp),
)
