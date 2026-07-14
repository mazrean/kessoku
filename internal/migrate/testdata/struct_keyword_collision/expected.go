//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package struct_keyword_collision

import (
	"github.com/mazrean/kessoku"
)

var ConfigSet = kessoku.Set(
	kessoku.Provide(func(type_ string, type__2 string) Config {
		return Config{Type: type_, Type_: type__2}
	}),
	kessoku.Provide(func(type_ string, type__2 string) *Config {
		return &Config{Type: type_, Type_: type__2}
	}),
)
