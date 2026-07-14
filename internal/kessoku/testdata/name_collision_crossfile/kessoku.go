package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// This file uses an injector name that already exists in types.go (cross-file)
// to verify that kessoku detects the collision and fails with a clear error message.
var _ = kessoku.Inject[*Foo](
	"NewFoo",
	kessoku.Provide(NewFooImpl),
)
