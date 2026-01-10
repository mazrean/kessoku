package main

//go:generate go tool kessoku $GOFILE

import (
	"io"
	"os"

	"github.com/mazrean/kessoku"
)

// Test Bind combined with Value - interface binding to a constant value
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Bind[io.Writer](kessoku.Value(os.Stdout)),
	kessoku.Provide(NewService),
)
