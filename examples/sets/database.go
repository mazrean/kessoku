package main

import "fmt"

// Database represents a database connection
type Database struct {
	connectionString string
}

// NewDatabase creates a new database connection
func NewDatabase(config *Config) (*Database, error) {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.DatabaseName,
	)

	return &Database{
		connectionString: connectionString,
	}, nil
}
