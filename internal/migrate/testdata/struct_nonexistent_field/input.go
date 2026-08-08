package struct_nonexistent_field

import (
	"github.com/google/wire"
)

type Config struct {
	Host string
}

var ConfigSet = wire.NewSet(wire.Struct(new(Config), "NonExistent", "Host"))
