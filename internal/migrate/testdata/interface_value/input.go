package interface_value

import (
	"github.com/google/wire"
)

type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {}

var logValue = &ConsoleLogger{}

var LoggerSet = wire.NewSet(wire.InterfaceValue(new(Logger), logValue))
