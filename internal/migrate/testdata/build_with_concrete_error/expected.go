//go:generate go tool kessoku $GOFILE

package build_with_concrete_error

import (
	"github.com/mazrean/kessoku"
)

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Provide(NewApp),
)
