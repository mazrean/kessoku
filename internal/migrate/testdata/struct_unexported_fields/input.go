package struct_unexported_fields

import (
	"github.com/google/wire"
)

// Config has both exported and unexported fields.
// wire.Struct with "*" should only inject exported fields.
type Config struct {
	Host     string
	Port     int
	password string
}

var ConfigSet = wire.NewSet(wire.Struct(new(Config), "*"))
