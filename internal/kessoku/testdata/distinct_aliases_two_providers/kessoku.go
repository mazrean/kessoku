//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Verify that two distinct aliases sharing the same underlying type (string)
// can each have their own provider without triggering a "multiple providers"
// conflict error. Before the alias-key fix, both aliases resolved to the
// same key ("string"), so the second provider would collide with the first.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDBConnStr),
	kessoku.Provide(NewCacheConnStr),
	kessoku.Provide(NewDBClient),
	kessoku.Provide(NewCacheClient),
	kessoku.Provide(NewApp),
)
