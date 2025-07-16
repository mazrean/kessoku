package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Example 1: Basic Set - group related providers
var _ = kessoku.Inject[*App](
	"InitializeAppBasic",
	kessoku.Set(
		kessoku.Provide(NewConfig),
		kessoku.Provide(NewDatabase),
	),
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)

// Example 2: Reusable Set - define once, use multiple times
var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

var _ = kessoku.Inject[*App](
	"InitializeAppWithSet",
	DatabaseSet, // Reuse the set
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)

// Example 3: Nested Sets - use sets inside other sets
var ServiceSet = kessoku.Set(
	DatabaseSet, // Include another set
	kessoku.Provide(NewUserService),
)

var _ = kessoku.Inject[*App](
	"InitializeAppWithNestedSets",
	ServiceSet, // This includes both database and service
	kessoku.Provide(NewApp),
)