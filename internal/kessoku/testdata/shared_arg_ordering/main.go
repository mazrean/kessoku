package main

import "context"

// Config holds application configuration.
type Config struct {
	DSN string
}

// DB represents a database connection.
type DB struct {
	config *Config
}

// NewDB creates a new database. It requires context and config as external arguments.
func NewDB(ctx context.Context, cfg *Config) (*DB, error) {
	return &DB{config: cfg}, nil
}

// Server depends on DB and Config. Both come from intermediate provider (NewDB)
// and as a direct external argument respectively. This exercises the scenario
// where a shared external argument (*Config) is discovered first via the
// top-level provider (NewServer) BFS traversal before context.Context which is
// only needed by the lower-level provider (NewDB). The generated signature must
// preserve the declaration-relative order: providers with lower DeclOrder first.
type Server struct {
	db  *DB
	cfg *Config
}

// NewServer creates a new server. It takes *DB (from NewDB) and *Config (external arg).
func NewServer(db *DB, cfg *Config) *Server {
	return &Server{db: db, cfg: cfg}
}

func main() {}
