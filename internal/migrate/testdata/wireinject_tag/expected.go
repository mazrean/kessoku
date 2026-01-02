//go:generate go tool kessoku $GOFILE

package wireinjecttag

import (
	"github.com/mazrean/kessoku"
)

var TestSet = kessoku.Set(
	kessoku.Provide(NewFoo),
)
