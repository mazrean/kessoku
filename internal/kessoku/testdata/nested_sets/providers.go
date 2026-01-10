package main

// Config holds application configuration.
type Config struct {
	Name string
}

// NewConfig creates a new configuration.
func NewConfig() *Config {
	return &Config{Name: "app"}
}

// Logger provides logging functionality.
type Logger struct{}

// NewLogger creates a new logger.
func NewLogger() *Logger {
	return &Logger{}
}

// Database represents a database connection.
type Database struct {
	config *Config
	logger *Logger
}

// NewDatabase creates a new database with dependencies.
func NewDatabase(config *Config, logger *Logger) *Database {
	return &Database{config: config, logger: logger}
}

// Cache represents a cache layer.
type Cache struct {
	db *Database
}

// NewCache creates a new cache with database dependency.
func NewCache(db *Database) *Cache {
	return &Cache{db: db}
}

// App is the main application.
type App struct {
	cache *Cache
}

// NewApp creates a new app with all dependencies.
func NewApp(cache *Cache) *App {
	return &App{cache: cache}
}
