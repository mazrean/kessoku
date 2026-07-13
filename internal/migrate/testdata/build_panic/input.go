//go:build wireinject

package build_panic

import "github.com/google/wire"

type App struct {
	DB *DB
}

type DB struct{}

func NewDB() *DB {
	return &DB{}
}

func NewApp(db *DB) *App {
	return &App{DB: db}
}

func InitializeApp() *App {
	panic(wire.Build(NewDB, NewApp))
}
