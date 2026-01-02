package merge_complex

import (
	"github.com/google/wire"
)

type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {}

type Service struct {
	db     *DB
	logger Logger
}

func NewService(db *DB, logger Logger) *Service {
	return &Service{db: db, logger: logger}
}

var ServiceSet = wire.NewSet(
	NewService,
	wire.Bind(new(Logger), new(*ConsoleLogger)),
	NewConsoleLogger,
	DBSet,
)

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}
