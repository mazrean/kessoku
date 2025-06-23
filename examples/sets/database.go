package main

import "fmt"

// Database represents a database connection
type Database struct {
	connectionString string
}

// NewDatabase creates a new database connection
func NewDatabase(config *Config) (*Database, error) {
	connectionString := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s",
		config.Database.Host,
		config.Database.Port,
		config.Database.Database,
		config.Database.Username,
		config.Database.Password,
	)

	return &Database{
		connectionString: connectionString,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	fmt.Println("Database connection closed")
	return nil
}