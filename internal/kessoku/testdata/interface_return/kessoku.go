package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test provider that returns interface type directly
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewRepository),
	kessoku.Provide(NewService),
)
