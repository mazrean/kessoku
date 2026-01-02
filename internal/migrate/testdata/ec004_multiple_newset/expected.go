package ec004_multiple_newset

import (
	"github.com/mazrean/kessoku"
)

var FooSet = kessoku.Set(kessoku.Provide(NewFoo))
var BarSet = kessoku.Set(kessoku.Provide(NewBar))
