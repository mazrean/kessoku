package main

import "github.com/mazrean/kessoku"

// Demonstrates kessoku.As for single interface binding
var _ = kessoku.Inject[*UserService](
	"InitializeUserService",
	// Bind concrete implementations to interfaces using kessoku.As
	kessoku.As[UserRepository](kessoku.Provide(NewDatabaseUserRepository)),
	kessoku.As[Logger](kessoku.Provide(NewConsoleLogger)),
	kessoku.Provide(NewUserService),
)