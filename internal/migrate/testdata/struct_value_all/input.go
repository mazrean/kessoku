//go:build wireinject

package struct_value_all

import (
	"github.com/google/wire"
)

type Config struct {
	Host string
	Port int
}

// wire.Struct(new(Config)) should provide both Config (value) and *Config (pointer).
var ConfigSet = wire.NewSet(wire.Struct(new(Config), "*"))
