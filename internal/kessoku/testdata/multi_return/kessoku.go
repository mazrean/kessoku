package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test provider that returns multiple values (not just error)
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfigs),
	kessoku.Provide(NewService),
)
