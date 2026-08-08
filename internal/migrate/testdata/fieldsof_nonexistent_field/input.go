package fieldsof_nonexistent_field

import (
	"github.com/google/wire"
)

type Config struct {
	Host string
}

var ConfigSet = wire.NewSet(wire.FieldsOf(new(Config), "Host", "Timeout"))
