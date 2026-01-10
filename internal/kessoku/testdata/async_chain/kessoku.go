package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test chain of async providers: Config -> Database -> Cache -> App
// Each depends on the previous one
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Async(kessoku.Provide(NewDatabase)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Provide(NewApp),
)
