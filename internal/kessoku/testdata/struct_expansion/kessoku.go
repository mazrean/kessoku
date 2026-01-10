//go:generate go run ../../cmd/kessoku kessoku.go

// Package main demonstrates the usage of kessoku.Struct[T]() annotation
// for automatic struct field expansion as dependencies.
package main

import (
	"github.com/mazrean/kessoku"
)

// Config holds database configuration fields.
// When used with kessoku.Struct[*Config](), all exported fields
// become available as individual dependencies for injection.
type Config struct {
	DBHost string
	DBPort int
	Debug  bool
}

// Default values for configuration.
const (
	defaultDBHost = "localhost"
	defaultDBPort = 5432 //nolint:mnd // example code
	defaultDebug  = true
)

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		DBHost: defaultDBHost,
		DBPort: defaultDBPort,
		Debug:  defaultDebug,
	}
}

// Database represents a database connection.
type Database struct {
	host  string
	port  int
	debug bool
}

// NewDatabase creates a new Database with the given host and port.
// It receives string and int from Config's exported fields via struct expansion.
func NewDatabase(host string, port int, debug bool) *Database {
	return &Database{
		host:  host,
		port:  port,
		debug: debug,
	}
}

// InitializeDatabase is the generated injector function.
// The kessoku.Struct[*Config]() annotation automatically expands
// Config's exported fields (DBHost, DBPort, Debug) as individual
// dependencies, which are then passed to NewDatabase.
var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewDatabase),
)
