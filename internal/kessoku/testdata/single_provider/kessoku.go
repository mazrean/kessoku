package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test minimal case: just one provider
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewService),
)
