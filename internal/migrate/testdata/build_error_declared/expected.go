//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package build_error_declared

import (
	"github.com/mazrean/kessoku"
)

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Value((error)(nil)),
	kessoku.Provide(NewApp),
)
