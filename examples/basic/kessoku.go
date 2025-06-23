package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// InitializeApp creates and initializes the application with all dependencies.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewLogger),
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)
