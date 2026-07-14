//go:build wireinject

package build_error_declared

import "github.com/google/wire"

type App struct {
	name string
}

func NewApp(name string) *App {
	return &App{name: name}
}

// InitApp declares error return even though no provider returns error.
func InitApp(name string) (*App, error) {
	wire.Build(NewApp)
	return nil, nil
}
