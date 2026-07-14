package main

type Config struct {
	DSN string
}

func NewConfig() *Config {
	return &Config{DSN: "postgres://localhost/db"}
}

type DB struct {
	config *Config
}

func NewDB(config *Config) (*DB, error) {
	return &DB{config: config}, nil
}

type Cache struct {
	config *Config
}

func NewCache(config *Config) *Cache {
	return &Cache{config: config}
}

type App struct {
	db    *DB
	cache *Cache
}

func NewApp(db *DB, cache *Cache) *App {
	return &App{db: db, cache: cache}
}

func main() {}
