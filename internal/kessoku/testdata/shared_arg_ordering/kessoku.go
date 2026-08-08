package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// InitServer injects NewServer which depends on NewDB.
// Both NewDB and NewServer require *Config as an external argument.
// NewDB also requires context.Context.
// Without the fix, BFS discovers *Config (via NewServer.Requires[1]) before
// context.Context (via NewDB.Requires[0]), producing the wrong signature:
//
//	func InitServer(cfg *Config, ctx context.Context) *Server
//
// The correct signature must order args by (DeclOrder, requiresIndex) of the
// provider that first introduces each arg:
//
//	func InitServer(ctx context.Context, cfg *Config) *Server
var _ = kessoku.Inject[*Server](
	"InitServer",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewServer),
)
