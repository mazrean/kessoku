package main

// Config holds application configuration
type Config struct {
	Port     string
	Database DatabaseConfig
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		Port: "8080",
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			Database: "myapp",
			Username: "user",
			Password: "password",
		},
	}
}