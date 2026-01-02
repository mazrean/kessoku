package struct_fields

import (
	"github.com/google/wire"
)

type Config struct {
	Host    string
	Port    int
	Timeout int
}

var ConfigSet = wire.NewSet(wire.Struct(new(Config), "Host", "Port"))
