//go:generate go tool kessoku $GOFILE

package merge

import (
	"github.com/mazrean/kessoku"
)

var FooSet = kessoku.Set(
	kessoku.Provide(NewFoo),
)
var BarSet = kessoku.Set(
	kessoku.Provide(NewBar),
)
