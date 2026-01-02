//go:generate go tool kessoku $GOFILE

package value

import (
	"github.com/mazrean/kessoku"
)

var ConfigSet = kessoku.Set(
	kessoku.Value("config-value"),
)
