package struct_keyword_field

import (
	"github.com/google/wire"
)

// EventLog has a field named "Type" whose toLowerCamel result is the Go keyword "type".
type EventLog struct {
	Type string
	Name string
}

var Set = wire.NewSet(wire.Struct(new(EventLog), "*"))
