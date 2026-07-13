//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// SyncService has no deps and goes to pool 0 (main goroutine).
// AsyncService also has no deps but must run in a dedicated goroutine,
// not serialised behind SyncService in pool 0.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewSyncService),
	kessoku.Async(kessoku.Provide(NewAsyncService)),
	kessoku.Provide(NewApp),
)
