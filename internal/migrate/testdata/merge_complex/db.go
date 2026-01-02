package merge_complex

import (
	"github.com/google/wire"
)

type DBConfig struct {
	Host string
	Port int
}

type DB struct {
	config *DBConfig
}

func NewDB(config *DBConfig) *DB {
	return &DB{config: config}
}

var dbConfig = &DBConfig{Host: "localhost", Port: 5432}

var DBSet = wire.NewSet(
	NewDB,
	wire.Value(dbConfig),
)
