package struct_all

import (
	"github.com/google/wire"
)

type Config struct {
	Host string
	Port int
}

var ConfigSet = wire.NewSet(wire.Struct(new(Config), "*"))
