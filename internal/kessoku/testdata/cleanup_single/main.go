package main

// DB represents a database connection.
type DB struct{}

// NewDB creates a new DB and returns a cleanup function to close it.
func NewDB() (*DB, func()) {
	return &DB{}, func() {}
}

// App uses the database.
type App struct {
	db *DB
}

// NewApp creates a new App.
func NewApp(db *DB) *App {
	return &App{db: db}
}

func main() {}
