//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// DB is the concrete database struct.
type DB struct{}

// Database is a type alias for DB.
type Database = DB

// NewDB creates a DB instance.
func NewDB() *DB {
	return &DB{}
}

// App depends on *Database (an alias for *DB).
type App struct{}

// NewApp creates an App given a *Database (alias for *DB).
func NewApp(db *Database) *App {
	return &App{}
}

// Test that the alias *Database and concrete *DB are treated as identical types.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewApp),
)
