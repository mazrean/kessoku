package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test multiple async providers that all return errors
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Async(kessoku.Provide(NewDatabase)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Async(kessoku.Provide(NewMessaging)),
	kessoku.Provide(NewApp),
)
