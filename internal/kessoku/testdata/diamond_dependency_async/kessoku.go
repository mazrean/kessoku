package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test diamond dependency pattern with async: App depends on DB (async) and Cache (sync),
// both depend on Config. DB and Cache should be initialized in parallel.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Async(kessoku.Provide(NewDB)),
	kessoku.Provide(NewCache),
	kessoku.Provide(NewApp),
)
