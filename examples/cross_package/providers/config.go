// Package providers contains shared provider functions for cross-package dependency injection.
package providers

// ExternalConfig represents configuration from another package.
type ExternalConfig struct {
	DatabaseURL string
	APIKey      string
}

// NewExternalConfig creates a new external configuration.
func NewExternalConfig() *ExternalConfig {
	return &ExternalConfig{
		DatabaseURL: "postgres://localhost/testdb",
		APIKey:      "secret-api-key",
	}
}
