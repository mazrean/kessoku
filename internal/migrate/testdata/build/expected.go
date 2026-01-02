//go:generate go tool kessoku $GOFILE

package build

import (
	"github.com/mazrean/kessoku"
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewApp),
)
