package main

import (
	"database/sql"
	"fmt"
)

// Database represents a database connection.
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection.
// wire: provider
func NewDatabase(config *Config) (*Database, error) {
	conn, err := sql.Open("sqlite3", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Database{conn: conn}, nil
}

// Close closes the database connection.
func (db *Database) Close() error {
	return db.conn.Close()
}

// Query executes a query.
func (db *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}
