//go:generate go tool kessoku $GOFILE

package cross_pkg_set_ref

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/migrate/testdata/cross_pkg_set_ref/pkg"
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	pkg.StorerSet,
	kessoku.Provide(NewApp),
)
