package struct_keyword_collision

import (
	"github.com/google/wire"
)

type Config struct {
	Type  string
	Type_ string
}

var ConfigSet = wire.NewSet(wire.Struct(new(Config), "*"))
