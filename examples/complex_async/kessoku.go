package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*App](
	"InitializeComplexApp",
	// First level: Config service (independent)
	kessoku.Provide(NewConfig),
	
	// Second level: Basic services (depend on config, can run in parallel)
	kessoku.Async(kessoku.Provide(NewDatabaseService)),
	kessoku.Async(kessoku.Provide(NewCacheService)),
	kessoku.Async(kessoku.Provide(NewMessagingService)),
	
	// Third level: Higher-level services (depend on basic services)
	kessoku.Async(kessoku.Provide(NewUserService)),      // depends on DatabaseService
	kessoku.Async(kessoku.Provide(NewSessionService)),   // depends on CacheService
	
	// Fourth level: Notification service (depends on all user-facing services)
	kessoku.Async(kessoku.Provide(NewNotificationService)), // depends on UserService, SessionService, MessagingService
	
	// Final level: App
	kessoku.Provide(NewComplexApp), // depends on NotificationService
)