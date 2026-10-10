//go:build wireinject

package cross_pkg_set_ref_newset

import (
	"github.com/google/wire"
	store "github.com/mazrean/kessoku/internal/migrate/testdata/cross_pkg_set_ref_newset/storage"
)

// Service uses the database.
type Service struct {
	db *store.DB
}

// NewService creates a new Service.
func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

// Logger is a local dependency.
type Logger struct{}

// NewLogger creates a new Logger.
func NewLogger() *Logger {
	return &Logger{}
}

// DBSet is a local set that shares its name with store.DBSet; the two must
// not be confused.
var DBSet = wire.NewSet(NewLogger)

// ServiceSet references both the external and the local DBSet.
var ServiceSet = wire.NewSet(store.DBSet, DBSet, NewService)

func InitializeService() *Service {
	wire.Build(ServiceSet)
	return nil
}
