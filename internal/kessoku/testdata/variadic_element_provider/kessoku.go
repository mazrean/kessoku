//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Option represents a service option.
type Option struct {
	Name string
}

// NewOption creates a single option.
func NewOption() Option {
	return Option{Name: "default"}
}

// Service represents a service.
type Service struct {
	opts []Option
}

// NewService creates a service with variadic options.
func NewService(opts ...Option) *Service {
	return &Service{opts: opts}
}

// An element-type provider (NewOption returns Option) should satisfy the
// variadic last parameter of NewService (...Option) without creating an
// external []Option argument.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewOption),
	kessoku.Provide(NewService),
)
