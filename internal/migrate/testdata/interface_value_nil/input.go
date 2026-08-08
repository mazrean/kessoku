//go:build wireinject

package interface_value_nil

import (
	"github.com/google/wire"
)

type Logger interface {
	Log(msg string)
}

var LoggerSet = wire.NewSet(wire.InterfaceValue(new(Logger), nil))
