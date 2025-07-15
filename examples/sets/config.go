package main

// Config holds database configuration
type Config struct {
	Host         string
	Port         string
	DatabaseName string
	Username     string
	Password     string
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		Host:         "localhost",
		Port:         "5432",
		DatabaseName: "myapp",
		Username:     "user",
		Password:     "password",
	}
}
