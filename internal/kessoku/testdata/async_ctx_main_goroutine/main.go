package main

import "context"

// NewDatabase is context-aware and async; the graph places it on the main goroutine
// because it is topologically first when all providers are async.
func NewDatabase(ctx context.Context) (*Database, error) {
	return &Database{}, nil
}

type Database struct{}

// NewCache is context-aware and async; the graph places it in an errgroup goroutine.
func NewCache(ctx context.Context) (*Cache, error) {
	return &Cache{}, nil
}

type Cache struct{}

type App struct {
	db    *Database
	cache *Cache
}

func NewApp(db *Database, cache *Cache) *App {
	return &App{db: db, cache: cache}
}

func main() {}
