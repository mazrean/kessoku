package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// This file intentionally creates a circular dependency to verify that
// the code generator detects cycles and fails with a clear error message.
var _ = kessoku.Inject[*A](
	"InitializeA",
	kessoku.Provide(NewA),
	kessoku.Provide(NewB),
)
