package importreplace

import (
	"github.com/mazrean/kessoku"
)

var MySet = kessoku.Set(
	kessoku.Provide(NewPrinter),
)
