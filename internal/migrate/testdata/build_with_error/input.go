//go:build wireinject

package build_with_error

import "github.com/google/wire"

type App struct {
	DB *DB
}

type DB struct{}

func NewDB() (*DB, error) {
	return &DB{}, nil
}

func NewApp(db *DB) *App {
	return &App{DB: db}
}

func InitializeApp() (*App, error) {
	wire.Build(NewDB, NewApp)
	return nil, nil
}
