package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test async providers combined with interface binding
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Async(kessoku.Bind[Repository](kessoku.Provide(NewDatabaseRepo))),
	kessoku.Provide(NewService),
)
