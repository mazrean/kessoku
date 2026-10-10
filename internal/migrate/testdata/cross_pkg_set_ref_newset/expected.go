//go:generate go tool kessoku $GOFILE

package cross_pkg_set_ref_newset

import (
	"github.com/mazrean/kessoku"
	store "github.com/mazrean/kessoku/internal/migrate/testdata/cross_pkg_set_ref_newset/storage"
)

var DBSet = kessoku.Set(
	kessoku.Provide(NewLogger),
)
var ServiceSet = kessoku.Set(
	store.DBSet,
	DBSet,
	kessoku.Provide(NewService),
)
var _ = kessoku.Inject[*Service](
	"InitializeService",
	ServiceSet,
)
