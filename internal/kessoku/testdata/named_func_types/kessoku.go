package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Providers returning named func types must not be rejected as wire-style
// cleanup functions, even when the underlying type is func() or func() error.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewShutdown),
	kessoku.Provide(NewCommit),
	kessoku.Provide(NewApp),
)
