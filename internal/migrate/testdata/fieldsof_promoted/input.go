package fieldsof_promoted

import (
	"github.com/google/wire"
)

type Inner struct {
	DB   *string
	Name string
}

type Config struct {
	Inner
	Port int
}

var FieldsSet = wire.NewSet(wire.FieldsOf(new(Config), "DB"))
