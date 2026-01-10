package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test diamond dependency pattern: App depends on ServiceA and ServiceB, both depend on Database
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewServiceA),
	kessoku.Provide(NewServiceB),
	kessoku.Provide(NewApp),
)
