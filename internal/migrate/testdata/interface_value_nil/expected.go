//go:generate go tool kessoku $GOFILE

package interface_value_nil

import (
	"github.com/mazrean/kessoku"
)

var LoggerSet = kessoku.Set(
	kessoku.Bind[Logger](kessoku.Value[Logger](nil)))
