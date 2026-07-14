//go:build wireinject

package build_alias_return

import "github.com/google/wire"

type App struct{}

// AppAlias is a type alias for App.
type AppAlias = App

func NewApp() *App {
	return &App{}
}

func InitApp() *AppAlias {
	wire.Build(NewApp)
	return nil
}
