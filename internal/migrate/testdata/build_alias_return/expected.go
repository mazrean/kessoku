//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package build_alias_return

import (
	"github.com/mazrean/kessoku"
)

var _ = kessoku.Inject[*AppAlias](
	"InitApp",
	kessoku.Provide(NewApp),
)
