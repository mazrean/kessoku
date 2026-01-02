//go:generate go tool kessoku $GOFILE

package struct_fields

import (
	"github.com/mazrean/kessoku"
)

var ConfigSet = kessoku.Set(
	kessoku.Provide(func(host string, port int) *Config {
		return &Config{Host: host, Port: port}
	}),
)
