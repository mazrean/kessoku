//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package build_crossfile_error

import (
	"github.com/mazrean/kessoku"
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	DBSet,
	kessoku.Provide(NewApp),
)
var DBSet = kessoku.Set(
	kessoku.Provide(NewDB),
)
