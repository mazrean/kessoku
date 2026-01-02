package value

import "github.com/google/wire"

var ConfigSet = wire.NewSet(
	wire.Value("config-value"),
)
