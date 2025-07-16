package main

import (
	"fmt"
	"log"
)

// Database represents a simple data store.
type Database struct {
	config *Config
	users  map[int]*User
}

// NewDatabase creates a new database connection.
func NewDatabase(config *Config) (*Database, error) {
	if config.Debug {
		log.Println("Creating database connection")
	}
	
	// Initialize with some sample data
	users := map[int]*User{
		1: {ID: 1, Name: "Alice", Email: "alice@example.com"},
		2: {ID: 2, Name: "Bob", Email: "bob@example.com"},
	}

	return &Database{
		config: config,
		users:  users,
	}, nil
}

// GetUser retrieves a user by ID.
func (db *Database) GetUser(id int) (*User, error) {
	if db.config.Debug {
		log.Printf("Querying user with ID: %d", id)
	}
	
	user, exists := db.users[id]
	if !exists {
		return nil, fmt.Errorf("user with ID %d not found", id)
	}
	
	return user, nil
}

