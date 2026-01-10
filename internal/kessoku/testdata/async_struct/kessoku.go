//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Config holds application configuration.
type Config struct {
	APIKey    string
	CacheTTL  int
	DebugMode bool
}

// NewConfig creates a new configuration (simulates slow config loading).
func NewConfig() *Config {
	return &Config{
		APIKey:    "secret-key",
		CacheTTL:  300,
		DebugMode: true,
	}
}

// Service uses expanded config fields.
type Service struct {
	key   string
	ttl   int
	debug bool
}

// NewService creates a new service with config fields.
func NewService(key string, ttl int, debug bool) *Service {
	return &Service{key: key, ttl: ttl, debug: debug}
}

// Test Async with Struct provider - async struct field expansion
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Async(kessoku.Provide(NewConfig)),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewService),
)
