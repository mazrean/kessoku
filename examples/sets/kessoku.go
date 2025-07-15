//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Example 1: Basic usage without Sets (for comparison)
var _ = kessoku.Inject[*App](
	"InitializeAppBasic",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)

// Example 2: Using inline kessoku.Set to group related providers
var _ = kessoku.Inject[*App](
	"InitializeAppWithInlineSet",
	kessoku.Set(
		kessoku.Provide(NewConfig),
		kessoku.Provide(NewDatabase),
	),
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)

// DatabaseSet groups database-related providers together
var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

// Example 3: Using Set variable for reusability
var _ = kessoku.Inject[*App](
	"InitializeAppWithSetVariable",
	DatabaseSet, // Reuse pre-defined set
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)

// ServiceSet demonstrates nested Sets - using one Set inside another
var ServiceSet = kessoku.Set(
	DatabaseSet, // Include the database set
	kessoku.Provide(NewUserService),
)

var _ = kessoku.Inject[*App](
	"InitializeAppWithNestedSets",
	ServiceSet, // This includes both database and service
	kessoku.Provide(NewApp),
)
