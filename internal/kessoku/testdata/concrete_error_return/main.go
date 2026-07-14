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

// NewDatabase returns a concrete *MyError alongside *Database. *MyError
// implements the error interface but is NOT the error interface type itself,
// so kessoku must treat it as a normal provided type — not as the error
// return slot. Since no other provider requires *MyError here, its value is
// discarded (blank identifier) in the generated code.
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
