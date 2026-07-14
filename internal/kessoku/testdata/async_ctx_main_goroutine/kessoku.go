package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Both providers are async and context-aware.
// The graph places NewDatabase on the main goroutine (first topological pool)
// and NewCache in an errgroup goroutine. If NewCache fails, errgroup cancels
// the context. The main-goroutine error handler must use context.Cause(ctx) to
// surface the root cause instead of context.Canceled.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Async(kessoku.Provide(NewDatabase)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Provide(NewApp),
)
