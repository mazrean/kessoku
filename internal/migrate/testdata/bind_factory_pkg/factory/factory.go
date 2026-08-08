package factory

import "github.com/mazrean/kessoku/internal/migrate/testdata/bind_factory_pkg/impl"

// NewDB creates a new DB instance.
func NewDB() *impl.DB {
	return &impl.DB{}
}
