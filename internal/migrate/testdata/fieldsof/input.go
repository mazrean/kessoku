package fieldsof

import (
	"github.com/google/wire"
)

type DBConn struct{}
type CacheConn struct{}

type Config struct {
	DB    *DBConn
	Cache *CacheConn
}

var FieldsSet = wire.NewSet(wire.FieldsOf(new(Config), "DB", "Cache"))
