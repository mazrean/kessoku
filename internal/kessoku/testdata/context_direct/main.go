package main

import "context"

type Config struct {
	DSN string
}

func NewConfig() *Config {
	return &Config{DSN: "test-dsn"}
}

type Database struct {
	config *Config
}

// NewDatabase takes context.Context directly (not via Async)
func NewDatabase(ctx context.Context, config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

func main() {
}
