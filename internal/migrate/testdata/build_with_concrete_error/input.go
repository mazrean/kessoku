//go:build wireinject

package build_with_concrete_error

import "github.com/google/wire"

type App struct{}

// MyError is a concrete error type (not the error interface itself).
type MyError struct {
	Msg string
}

func (e *MyError) Error() string { return e.Msg }

// NewApp returns (*App, *MyError). anyProviderReturnsError must recognise
// *MyError as implementing error via types.Implements, not just types.Identical.
func NewApp() (*App, *MyError) {
	return &App{}, nil
}

func InitApp() (*App, error) {
	wire.Build(NewApp)
	return nil, nil
}
