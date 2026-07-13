//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Test that multiple providers each returning func() do not collide and both
// cleanup functions are deferred.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewCache),
	kessoku.Provide(NewApp),
)
