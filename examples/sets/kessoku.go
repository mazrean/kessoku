//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// DatabaseSet groups database-related providers together
var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewDatabase),
)

// ServerSet groups server-related providers together  
var ServerSet = kessoku.Set(
	kessoku.Provide(NewServer),
)

// ServiceSet groups service-related providers together
var ServiceSet = kessoku.Set(
	DatabaseSet, // Use another set within a set
	kessoku.Provide(NewUserService),
)

// Example 1: Using inline kessoku.Set
var _ = kessoku.Inject[*App](
	"InitializeApp",
	// Inline Set usage - groups infrastructure providers
	kessoku.Set(
		kessoku.Provide(NewConfig),
		kessoku.Provide(NewDatabase),
		kessoku.Provide(NewServer),
	),
	// Service providers
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)

// Example 2: Using Set variables for better organization
var _ = kessoku.Inject[*App](
	"InitializeAppWithSets",
	kessoku.Provide(NewConfig), // Shared config
	ServerSet,  // Use pre-defined server set
	ServiceSet, // Use pre-defined service set (which includes DatabaseSet)
	kessoku.Provide(NewApp),
)

// Example 3: Mixing inline Sets, Set variables, and individual providers
var _ = kessoku.Inject[*UserService](
	"InitializeUserService",
	// Mix of inline Set and individual provider
	kessoku.Set(
		kessoku.Provide(NewConfig),
		kessoku.Provide(NewDatabase),
	),
	kessoku.Provide(NewUserService),
)

// CoreInfrastructureSet demonstrates nested Sets with complex organization
var CoreInfrastructureSet = kessoku.Set(
	kessoku.Set( // Nested inline Set
		kessoku.Provide(NewDatabase),
		kessoku.Provide(NewServer),
	),
)

var _ = kessoku.Inject[*App](
	"InitializeAppWithNestedSets",
	kessoku.Provide(NewConfig), // Shared config
	CoreInfrastructureSet, // Uses nested Sets
	kessoku.Provide(NewUserService),
	kessoku.Provide(NewApp),
)