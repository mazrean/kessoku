package main

// Config holds application configuration.
type Config struct {
	DatabaseURL string
	Port        int
}

// NewConfig creates a new configuration.
// wire: provider
func NewConfig() *Config {
	const defaultPort = 8080
	return &Config{
		DatabaseURL: "file::memory:?cache=shared",
		Port:        defaultPort,
	}
}
