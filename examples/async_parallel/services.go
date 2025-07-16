package main

import (
	"fmt"
	"time"
)

const (
	databaseConnectionDelay = 200 * time.Millisecond
	cacheConnectionDelay    = 150 * time.Millisecond
	messagingConnectionDelay = 180 * time.Millisecond
)

// DatabaseService simulates a database connection service
type DatabaseService struct {
	connectionString string
}

func NewDatabaseService() (*DatabaseService, error) {
	// Simulate slow database connection setup
	fmt.Println("Connecting to database...")
	time.Sleep(databaseConnectionDelay)
	fmt.Println("Database connected!")
	return &DatabaseService{
		connectionString: "postgres://localhost:5432/mydb",
	}, nil
}

// CacheService simulates a cache service
type CacheService struct {
	endpoint string
}

func NewCacheService() *CacheService {
	// Simulate slow cache connection setup
	fmt.Println("Connecting to cache...")
	time.Sleep(cacheConnectionDelay)
	fmt.Println("Cache connected!")
	return &CacheService{
		endpoint: "redis://localhost:6379",
	}
}

// MessagingService simulates a messaging service
type MessagingService struct {
	brokerURL string
}

func NewMessagingService() *MessagingService {
	// Simulate slow messaging setup
	fmt.Println("Connecting to message broker...")
	time.Sleep(messagingConnectionDelay)
	fmt.Println("Message broker connected!")
	return &MessagingService{
		brokerURL: "kafka://localhost:9092",
	}
}

// UserService depends on DatabaseService and CacheService
type UserService struct {
	db    *DatabaseService
	cache *CacheService
}

func NewUserService(db *DatabaseService, cache *CacheService) *UserService {
	fmt.Println("Initializing user service...")
	return &UserService{
		db:    db,
		cache: cache,
	}
}

// NotificationService depends on MessagingService
type NotificationService struct {
	messaging *MessagingService
}

func NewNotificationService(messaging *MessagingService) *NotificationService {
	fmt.Println("Initializing notification service...")
	return &NotificationService{
		messaging: messaging,
	}
}

// App depends on both UserService and NotificationService
type App struct {
	userService         *UserService
	notificationService *NotificationService
}

func NewApp(userService *UserService, notificationService *NotificationService) *App {
	fmt.Println("Initializing app...")
	return &App{
		userService:         userService,
		notificationService: notificationService,
	}
}

func (a *App) Run() {
	fmt.Println("App is running...")
	fmt.Println("- Database:", a.userService.db.connectionString)
	fmt.Println("- Cache:", a.userService.cache.endpoint)
	fmt.Println("- Messaging:", a.notificationService.messaging.brokerURL)
}
