//go:build wireinject

package struct_ptr_fields

import (
	"github.com/google/wire"
)

type Config struct {
	Host string
	Port int
}

var ConfigSet = wire.NewSet(wire.Struct(new(*Config), "Host", "Port"))
