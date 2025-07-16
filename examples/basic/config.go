package main

// Config holds application configuration.
type Config struct {
	AppName string
	Debug   bool
}

// NewConfig creates a new configuration.
func NewConfig() *Config {
	return &Config{
		AppName: "Kessoku Basic Example",
		Debug:   true,
	}
}