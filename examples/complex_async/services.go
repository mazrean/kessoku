package main

import (
	"fmt"
	"time"
)

// Configuration and basic services
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

// Higher-level services that depend on basic services
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

// Main application
type App struct {
	Notifications *NotificationService
	Ready         bool
}

// Provider functions

func NewConfig() *Config {
	fmt.Println("Creating config...")
	time.Sleep(100 * time.Millisecond)
	return &Config{
		DatabaseURL: "postgres://localhost:5432/db",
		CacheURL:    "redis://localhost:6379",
		MessageURL:  "rabbitmq://localhost:5672",
	}
}

func NewDatabaseService(config *Config) *DatabaseService {
	fmt.Printf("Creating database service with URL: %s\n", config.DatabaseURL)
	time.Sleep(200 * time.Millisecond)
	return &DatabaseService{
		Config: config,
		Ready:  true,
	}
}

func NewCacheService(config *Config) *CacheService {
	fmt.Printf("Creating cache service with URL: %s\n", config.CacheURL)
	time.Sleep(150 * time.Millisecond)
	return &CacheService{
		Config: config,
		Ready:  true,
	}
}

func NewMessagingService(config *Config) *MessagingService {
	fmt.Printf("Creating messaging service with URL: %s\n", config.MessageURL)
	time.Sleep(180 * time.Millisecond)
	return &MessagingService{
		Config: config,
		Ready:  true,
	}
}

func NewUserService(db *DatabaseService) *UserService {
	fmt.Println("Creating user service...")
	time.Sleep(120 * time.Millisecond)
	return &UserService{
		DB:    db,
		Ready: true,
	}
}

func NewSessionService(cache *CacheService) *SessionService {
	fmt.Println("Creating session service...")
	time.Sleep(100 * time.Millisecond)
	return &SessionService{
		Cache: cache,
		Ready: true,
	}
}

func NewNotificationService(users *UserService, sessions *SessionService, messaging *MessagingService) *NotificationService {
	fmt.Println("Creating notification service...")
	time.Sleep(80 * time.Millisecond)
	return &NotificationService{
		Users:     users,
		Sessions:  sessions,
		Messaging: messaging,
		Ready:     true,
	}
}

func NewComplexApp(notifications *NotificationService) *App {
	fmt.Println("Creating complex app...")
	time.Sleep(50 * time.Millisecond)
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