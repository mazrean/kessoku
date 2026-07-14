//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// DB is the concrete database struct.
type DB struct{}

// Database is a type alias for DB.
type Database = DB

// NewDB creates a Database instance (returning the alias type explicitly).
// After the alias-key fix, kessoku treats each alias as a distinct dependency
// key, so the provider must declare the alias in its return type for the
// dependency to be satisfied without an extra injector argument.
func NewDB() *Database {
	return &DB{}
}

// App depends on *Database (an alias for *DB).
type App struct{}

// NewApp creates an App given a *Database (alias for *DB).
func NewApp(db *Database) *App {
	return &App{}
}

// Test that a provider returning *Database satisfies a *Database parameter.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewApp),
)
