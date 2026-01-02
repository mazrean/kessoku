package build

import (
	"github.com/google/wire"
)

type App struct {
	db *DB
}

type DB struct{}

func NewApp(db *DB) *App {
	return &App{db: db}
}

func NewDB() *DB {
	return &DB{}
}

func InitializeApp() (*App, error) {
	wire.Build(NewDB, NewApp)
	return nil, nil
}
