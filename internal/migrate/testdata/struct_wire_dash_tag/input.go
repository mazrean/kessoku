package struct_wire_dash_tag

import (
	"github.com/google/wire"
)

// Config has a field marked with wire:"-" that wire skips during injection.
type Config struct {
	Host    string
	Port    int
	Version string `wire:"-"`
}

var ConfigSet = wire.NewSet(wire.Struct(new(Config), "*"))
