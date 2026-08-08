//go:build wireinject

package bind_factory_pkg

import (
	"github.com/google/wire"
	"github.com/mazrean/kessoku/internal/migrate/testdata/bind_factory_pkg/factory"
	"github.com/mazrean/kessoku/internal/migrate/testdata/bind_factory_pkg/impl"
)

// NewApp creates the application.
func NewApp(q impl.Querier) *App {
	return &App{q: q}
}

// App is the application struct.
type App struct {
	q impl.Querier
}

func InitializeApp() *App {
	wire.Build(
		factory.NewDB,
		wire.Bind(new(impl.Querier), new(*impl.DB)),
		NewApp,
	)
	return nil
}
