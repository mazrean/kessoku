//go:generate go tool kessoku $GOFILE

package basic

import (
	"github.com/mazrean/kessoku"
)

var SuperSet = kessoku.Set(
	kessoku.Provide(NewFoo),
)
