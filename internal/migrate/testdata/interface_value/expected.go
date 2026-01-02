package interface_value

import (
	"github.com/mazrean/kessoku"
)

var LoggerSet = kessoku.Set(kessoku.Bind[Logger](kessoku.Value(logValue)))
