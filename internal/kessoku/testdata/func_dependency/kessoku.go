//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// LoggerFunc is a function type for logging.
type LoggerFunc func(msg string)

// NewLoggerFunc creates a new logger function.
func NewLoggerFunc() LoggerFunc {
	return func(msg string) {
		// log implementation
	}
}

// Service uses a function type dependency.
type Service struct {
	logger LoggerFunc
}

// NewService creates a new service with a logger function.
func NewService(logger LoggerFunc) *Service {
	return &Service{logger: logger}
}

// Test function type as dependency
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewLoggerFunc),
	kessoku.Provide(NewService),
)
