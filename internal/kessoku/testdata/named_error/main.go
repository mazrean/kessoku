package main

// MyError is a named concrete type that implements the error interface but is
// NOT the error interface itself. kessoku must treat it as a normal provided
// type rather than the error return slot.
type MyError struct {
	Msg string
}

func (e *MyError) Error() string {
	return e.Msg
}

type Config struct {
	DSN string
}

// NewConfig returns (*Config, *MyError). *MyError implements error but is not
// the error interface, so it is treated as a normal provide type. Since no
// downstream provider requires *MyError, its value is discarded in the
// generated code (blank identifier).
func NewConfig() (*Config, *MyError) {
	return &Config{DSN: "test-dsn"}, nil
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}

func main() {
}
