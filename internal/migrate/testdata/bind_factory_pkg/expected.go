//go:generate go tool kessoku $GOFILE

package bind_factory_pkg

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/migrate/testdata/bind_factory_pkg/factory"
	"github.com/mazrean/kessoku/internal/migrate/testdata/bind_factory_pkg/impl"
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Bind[impl.Querier](kessoku.Provide(factory.NewDB)),
	kessoku.Provide(NewApp),
)
