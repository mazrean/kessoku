//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Declare async providers for services that can be initialized in parallel
var _ = kessoku.Inject[*App](
	"InitializeApp",
	// These three services can be initialized in parallel since they have no dependencies
	kessoku.Async(kessoku.Provide(NewDatabaseService)),
	kessoku.Async(kessoku.Provide(NewCacheService)),
	kessoku.Async(kessoku.Provide(NewMessagingService)),
	// UserService depends on database and cache, so it runs after the first group
	kessoku.Provide(NewUserService),
	// NotificationService depends on messaging, so it runs after the first group
	kessoku.Provide(NewNotificationService),
	// App depends on both services, so it runs last
	kessoku.Provide(NewApp),
)