package main

type Config struct {
	DSN      string
	CacheURL string
}

func NewConfig() (*Config, error) {
	return &Config{DSN: "test-dsn", CacheURL: "redis://localhost"}, nil
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type Cache struct {
	config *Config
}

func NewCache(config *Config) (*Cache, error) {
	return &Cache{config: config}, nil
}

type Service struct {
	db    *Database
	cache *Cache
}

func NewService(db *Database, cache *Cache) (*Service, error) {
	return &Service{db: db, cache: cache}, nil
}

type App struct {
	service *Service
}

func NewApp(service *Service) *App {
	return &App{service: service}
}

func main() {
}
