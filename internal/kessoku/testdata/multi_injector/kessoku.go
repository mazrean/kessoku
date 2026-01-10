//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Config is shared configuration.
type Config struct{}

// NewConfig creates a new configuration.
func NewConfig() *Config {
	return &Config{}
}

// Database is the database layer.
type Database struct{}

// NewDatabase creates a new database.
func NewDatabase(config *Config) (*Database, error) {
	return &Database{}, nil
}

// Cache is the cache layer.
type Cache struct{}

// NewCache creates a new cache.
func NewCache(config *Config) *Cache {
	return &Cache{}
}

// UserService depends on database.
type UserService struct{}

// NewUserService creates a user service.
func NewUserService(db *Database) *UserService {
	return &UserService{}
}

// CacheService depends on cache.
type CacheService struct{}

// NewCacheService creates a cache service.
func NewCacheService(cache *Cache) *CacheService {
	return &CacheService{}
}

// Test 1: Injector with error return
var _ = kessoku.Inject[*UserService](
	"InitializeUserService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
	kessoku.Provide(NewUserService),
)

// Test 2: Injector without error return
var _ = kessoku.Inject[*CacheService](
	"InitializeCacheService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewCache),
	kessoku.Provide(NewCacheService),
)

// Test 3: Simple injector
var _ = kessoku.Inject[*Config](
	"InitializeConfig",
	kessoku.Provide(NewConfig),
)
