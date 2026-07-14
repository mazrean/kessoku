package main

// ConfigProvider is an interface satisfied by *Config.
// When used with Bind[ConfigProvider](Struct[*Config]()), the interface
// should be resolved internally and not leaked as an external parameter.
type ConfigProvider interface {
	GetHost() string
	GetPort() int
}

// Config holds database configuration fields.
type Config struct {
	DBHost string
	DBPort int
}

func (c *Config) GetHost() string { return c.DBHost }
func (c *Config) GetPort() int    { return c.DBPort }

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		DBHost: "localhost",
		DBPort: 5432, //nolint:mnd // example code
	}
}

// Database represents a database connection.
type Database struct {
	host string
	port int
}

// NewDatabase creates a new Database using a ConfigProvider interface.
func NewDatabase(cfg ConfigProvider) *Database {
	return &Database{
		host: cfg.GetHost(),
		port: cfg.GetPort(),
	}
}

func main() {}
