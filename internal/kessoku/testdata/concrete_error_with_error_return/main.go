package main

// ValidationError is a concrete type implementing the error interface.
// A provider that returns (*ValidationError, error) must be accepted: the
// concrete type in the non-last position is a normal provided value, and
// only the bare error interface in the last position is the error slot.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

type Config struct {
	DSN string
}

func NewConfig() *Config {
	return &Config{DSN: "test-dsn"}
}

// Validate returns (*ValidationError, error). *ValidationError implements
// error but must NOT be treated as the error return slot; only the trailing
// error is the error slot.
func Validate(c *Config) (*ValidationError, error) {
	return nil, nil
}

type Service struct {
	config *Config
	verr   *ValidationError
}

func NewService(c *Config, verr *ValidationError) *Service {
	return &Service{config: c, verr: verr}
}

func main() {
}
