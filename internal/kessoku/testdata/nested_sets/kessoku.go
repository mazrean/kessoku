//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Test nested Sets: Sets that contain other Sets

// InnerSet groups low-level providers
var InnerSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewLogger),
)

// MiddleSet contains InnerSet and adds more providers
var MiddleSet = kessoku.Set(
	InnerSet,
	kessoku.Provide(NewDatabase),
)

// OuterSet contains MiddleSet (nested two levels deep)
var OuterSet = kessoku.Set(
	MiddleSet,
	kessoku.Provide(NewCache),
)

// Test deeply nested Sets with inline and variable Sets
var _ = kessoku.Inject[*App](
	"InitializeApp",
	OuterSet,
	kessoku.Provide(NewApp),
)
