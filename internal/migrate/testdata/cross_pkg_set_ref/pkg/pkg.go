package pkg

import "github.com/google/wire"

// Storer is an interface for storage operations.
type Storer interface {
	Store(key string, value []byte) error
}

// PostgresStorer is a concrete implementation of Storer.
type PostgresStorer struct{}

// Store stores a key-value pair.
func (p *PostgresStorer) Store(key string, value []byte) error {
	return nil
}

// NewPostgresStorer creates a new PostgresStorer.
func NewPostgresStorer() *PostgresStorer {
	return &PostgresStorer{}
}

// StorerSet is the wire set for storage providers.
var StorerSet = wire.NewSet(NewPostgresStorer)
