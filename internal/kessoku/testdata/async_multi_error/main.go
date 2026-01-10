package main

type Database struct{}

func NewDatabase() (*Database, error) {
	return &Database{}, nil
}

type Cache struct{}

func NewCache() (*Cache, error) {
	return &Cache{}, nil
}

type Messaging struct{}

func NewMessaging() (*Messaging, error) {
	return &Messaging{}, nil
}

type App struct {
	db        *Database
	cache     *Cache
	messaging *Messaging
}

func NewApp(db *Database, cache *Cache, messaging *Messaging) *App {
	return &App{db: db, cache: cache, messaging: messaging}
}

func main() {
}
