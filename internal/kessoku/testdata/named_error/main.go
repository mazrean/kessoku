package main

// MyError is a named concrete error type implementing the error interface.
type MyError struct {
	Msg string
}

func (e *MyError) Error() string {
	return e.Msg
}

type Config struct {
	DSN string
}

// NewConfig returns a named concrete error type instead of the error interface.
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
