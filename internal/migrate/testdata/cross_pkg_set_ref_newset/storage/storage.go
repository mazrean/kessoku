package storage

import "github.com/google/wire"

// DB is a database handle.
type DB struct{}

// NewDB creates a new DB.
func NewDB() *DB {
	return &DB{}
}

// DBSet is the wire set for database providers.
var DBSet = wire.NewSet(NewDB)
