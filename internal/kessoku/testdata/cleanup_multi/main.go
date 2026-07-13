package main

// DB represents a database connection.
type DB struct{}

// NewDB creates a new DB and returns a cleanup function to close it.
func NewDB() (*DB, func()) {
	return &DB{}, func() {}
}

// Cache represents a cache connection.
type Cache struct{}

// NewCache creates a new Cache and returns a cleanup function to close it.
func NewCache() (*Cache, func()) {
	return &Cache{}, func() {}
}

// App uses both the database and cache.
type App struct {
	db    *DB
	cache *Cache
}

// NewApp creates a new App.
func NewApp(db *DB, cache *Cache) *App {
	return &App{db: db, cache: cache}
}

func main() {}
