package main

type Config struct {
	DSN      string
	CacheURL string
}

func NewConfig() *Config {
	return &Config{DSN: "test-dsn", CacheURL: "redis://localhost"}
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type Cache struct {
	db *Database
}

func NewCache(db *Database) (*Cache, error) {
	return &Cache{db: db}, nil
}

type App struct {
	cache *Cache
}

func NewApp(cache *Cache) *App {
	return &App{cache: cache}
}

func main() {
}
