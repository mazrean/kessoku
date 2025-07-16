package main

import (
	"fmt"
	"time"
)

// DatabaseService simulates a slow database connection
type DatabaseService struct {
	Connected bool
}

func NewDatabaseService() (*DatabaseService, error) {
	fmt.Println("🔗 Connecting to database...")
	time.Sleep(200 * time.Millisecond) // Simulate slow connection
	fmt.Println("✅ Database connected!")
	return &DatabaseService{Connected: true}, nil
}

// CacheService simulates a slow cache connection
type CacheService struct {
	Connected bool
}

func NewCacheService() *CacheService {
	fmt.Println("🔗 Connecting to cache...")
	time.Sleep(150 * time.Millisecond) // Simulate slow connection
	fmt.Println("✅ Cache connected!")
	return &CacheService{Connected: true}
}

// MessagingService simulates a slow messaging service
type MessagingService struct {
	Connected bool
}

func NewMessagingService() *MessagingService {
	fmt.Println("🔗 Connecting to message broker...")
	time.Sleep(100 * time.Millisecond) // Simulate slow connection
	fmt.Println("✅ Message broker connected!")
	return &MessagingService{Connected: true}
}

// App combines all services
type App struct {
	database  *DatabaseService
	cache     *CacheService
	messaging *MessagingService
}

func NewApp(db *DatabaseService, cache *CacheService, messaging *MessagingService) *App {
	fmt.Println("🚀 Initializing app...")
	return &App{
		database:  db,
		cache:     cache,
		messaging: messaging,
	}
}

func (a *App) Run() {
	fmt.Println("\n🎉 App is running!")
	fmt.Printf("Database: %v | Cache: %v | Messaging: %v\n", 
		a.database.Connected, a.cache.Connected, a.messaging.Connected)
}