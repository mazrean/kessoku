//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Verify that when only a DBConnectionString provider exists (no CacheConnectionString
// provider), code generation fails rather than silently reusing the DB string for
// the cache client. Before the alias-key fix, both aliases resolved to the same
// key ("string"), so the single DBConnectionString provider was silently shared
// with the CacheConnectionString parameter, producing an incorrect wiring.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDBConnStr),
	kessoku.Provide(NewDBClient),
	kessoku.Provide(NewCacheClient),
	kessoku.Provide(NewApp),
)
