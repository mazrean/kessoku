package setpkg

import "github.com/mazrean/kessoku"

// Config holds configuration.
type Config struct {
	DSN string
}

// NewConfig creates a new Config.
func NewConfig() *Config {
	return &Config{DSN: "test"}
}

// ConfigSet groups configuration providers.
var ConfigSet = kessoku.Set(
	kessoku.Provide(NewConfig),
)
