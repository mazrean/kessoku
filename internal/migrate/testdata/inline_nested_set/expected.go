package inline_nested_set

import (
	"github.com/mazrean/kessoku"
)

var AllSet = kessoku.Set(
	kessoku.Provide(NewFoo),
	kessoku.Provide(NewBar),
)
