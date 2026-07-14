//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// InitializeService must take ctx as its FIRST parameter even though the
// provider requiring it is declared after the one requiring APIKey.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
