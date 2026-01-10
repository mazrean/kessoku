package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test multiple sync providers with error returns
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewCache),
	kessoku.Provide(NewService),
	kessoku.Provide(NewApp),
)
