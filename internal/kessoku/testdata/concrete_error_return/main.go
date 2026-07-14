package main

// MyError is a concrete error type (not the error interface itself).
type MyError struct {
	Code    int
	Message string
}

func (e *MyError) Error() string {
	return e.Message
}

type Config struct {
	DSN string
}

func NewConfig() *Config {
	return &Config{DSN: "test-dsn"}
}

type Database struct {
	config *Config
}

// NewDatabase returns a concrete *MyError instead of the error interface.
// When *MyError is nil, this must NOT trigger an error path.
func NewDatabase(config *Config) (*Database, *MyError) {
	return &Database{config: config}, nil
}

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

func main() {
}
