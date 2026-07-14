package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test that a provider returning (*ConcreteErrorType, error) is accepted:
// the concrete type implementing error is treated as a normal provided value,
// and only the trailing bare error interface is the error return slot.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(Validate),
	kessoku.Provide(NewService),
)
