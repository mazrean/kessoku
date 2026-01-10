package main

type Config struct {
	DSN string
}

func NewConfig() *Config {
	return &Config{DSN: "test-dsn"}
}

type Database struct {
	config *Config
}

// NewDatabase returns an error - this is a sync provider with error return
func NewDatabase(config *Config) (*Database, error) {
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
