//go:build wireinject

package cross_pkg_set_ref

import (
	"github.com/google/wire"
	"github.com/mazrean/kessoku/internal/migrate/testdata/cross_pkg_set_ref/pkg"
)

// App is the application struct.
type App struct {
	storer pkg.Storer
}

// NewApp creates a new App.
func NewApp(storer pkg.Storer) *App {
	return &App{storer: storer}
}

func InitializeApp() *App {
	wire.Build(pkg.StorerSet, wire.Bind(new(pkg.Storer), new(*pkg.PostgresStorer)), NewApp)
	return nil
}
