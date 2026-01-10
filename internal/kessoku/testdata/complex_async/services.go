package main

import (
	"fmt"
	"time"
)

const (
	configCreationDelay      = 100 * time.Millisecond
	databaseServiceDelay     = 200 * time.Millisecond
	cacheServiceDelay        = 150 * time.Millisecond
	messagingServiceDelay    = 180 * time.Millisecond
	userServiceDelay         = 120 * time.Millisecond
	sessionServiceDelay      = 100 * time.Millisecond
	notificationServiceDelay = 80 * time.Millisecond
	appCreationDelay         = 50 * time.Millisecond
)

// Config represents the application configuration.
type Config struct {
	DatabaseURL string
	CacheURL    string
	MessageURL  string
}

type DatabaseService struct {
	Config *Config
	Ready  bool
}

type CacheService struct {
	Config *Config
	Ready  bool
}

type MessagingService struct {
	Config *Config
	Ready  bool
}

// UserService provides user-related functionality and depends on basic services.
type UserService struct {
	DB    *DatabaseService
	Ready bool
}

type SessionService struct {
	Cache *CacheService
	Ready bool
}

type NotificationService struct {
	Users     *UserService
	Sessions  *SessionService
	Messaging *MessagingService
	Ready     bool
}

// App represents the main application that coordinates all services.
type App struct {
	Notifications *NotificationService
	Ready         bool
}

// Provider functions

func NewConfig() *Config {
	fmt.Println("Creating config...")
	time.Sleep(configCreationDelay)
	return &Config{
		DatabaseURL: "postgres://localhost:5432/db",
		CacheURL:    "redis://localhost:6379",
		MessageURL:  "rabbitmq://localhost:5672",
	}
}

func NewDatabaseService(config *Config) *DatabaseService {
	fmt.Printf("Creating database service with URL: %s\n", config.DatabaseURL)
	time.Sleep(databaseServiceDelay)
	return &DatabaseService{
		Config: config,
		Ready:  true,
	}
}

func NewCacheService(config *Config) *CacheService {
	fmt.Printf("Creating cache service with URL: %s\n", config.CacheURL)
	time.Sleep(cacheServiceDelay)
	return &CacheService{
		Config: config,
		Ready:  true,
	}
}

func NewMessagingService(config *Config) *MessagingService {
	fmt.Printf("Creating messaging service with URL: %s\n", config.MessageURL)
	time.Sleep(messagingServiceDelay)
	return &MessagingService{
		Config: config,
		Ready:  true,
	}
}

func NewUserService(db *DatabaseService) *UserService {
	fmt.Println("Creating user service...")
	time.Sleep(userServiceDelay)
	return &UserService{
		DB:    db,
		Ready: true,
	}
}

func NewSessionService(cache *CacheService) *SessionService {
	fmt.Println("Creating session service...")
	time.Sleep(sessionServiceDelay)
	return &SessionService{
		Cache: cache,
		Ready: true,
	}
}

func NewNotificationService(users *UserService, sessions *SessionService, messaging *MessagingService) *NotificationService {
	fmt.Println("Creating notification service...")
	time.Sleep(notificationServiceDelay)
	return &NotificationService{
		Users:     users,
		Sessions:  sessions,
		Messaging: messaging,
		Ready:     true,
	}
}

func NewComplexApp(notifications *NotificationService) *App {
	fmt.Println("Creating complex app...")
	time.Sleep(appCreationDelay)
	return &App{
		Notifications: notifications,
		Ready:         true,
	}
}

func (app *App) Run() {
	fmt.Println("Complex app is running!")
	fmt.Printf("Database ready: %t\n", app.Notifications.Users.DB.Ready)
	fmt.Printf("Cache ready: %t\n", app.Notifications.Sessions.Cache.Ready)
	fmt.Printf("Messaging ready: %t\n", app.Notifications.Messaging.Ready)
}
