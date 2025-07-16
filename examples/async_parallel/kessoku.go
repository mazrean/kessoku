package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Demonstrate parallel execution with kessoku.Async()
// Independent services run in parallel: 200ms + 150ms + 100ms = 200ms (max)
// Sequential would be: 200ms + 150ms + 100ms = 450ms (sum)
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Async(kessoku.Provide(NewDatabaseService)), // 200ms - runs in parallel
	kessoku.Async(kessoku.Provide(NewCacheService)),    // 150ms - runs in parallel
	kessoku.Async(kessoku.Provide(NewMessagingService)), // 100ms - runs in parallel
	kessoku.Provide(NewApp),                             // Waits for all async providers
)