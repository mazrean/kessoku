//go:generate go tool kessoku $GOFILE

package struct_value_all

import (
	"github.com/mazrean/kessoku"
)

var ConfigSet = kessoku.Set(
	kessoku.Provide(func(host string, port int) Config {
		return Config{Host: host, Port: port}
	}),
	kessoku.Provide(func(host string, port int) *Config {
		return &Config{Host: host, Port: port}
	}),
)
