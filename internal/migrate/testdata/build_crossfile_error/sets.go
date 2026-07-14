//go:build wireinject

package build_crossfile_error

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

var DBSet = wire.NewSet(NewDB)
