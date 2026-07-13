//go:generate go tool kessoku $GOFILE

package main

import (
	"context"

	"github.com/mazrean/kessoku"
)

// InitializeService has two async providers, one of which returns *Errgroup
// (a type whose lowerCamel name is "errgroup", same as the sync/errgroup pkg alias).
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Async(kessoku.Provide(NewErrgroup)),
	kessoku.Async(kessoku.Provide(NewDatabase)),
	kessoku.Provide(NewService),
)

func InitializeService(ctx context.Context) (*Service, error) {
	panic("kessoku: inject")
}
