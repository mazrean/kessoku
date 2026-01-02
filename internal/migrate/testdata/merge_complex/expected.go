//go:generate go tool kessoku $GOFILE

package merge_complex

import (
	"github.com/mazrean/kessoku"
)

var DBSet = kessoku.Set(
	kessoku.Provide(NewDB),
	kessoku.Value(dbConfig),
)
var ServiceSet = kessoku.Set(
	kessoku.Provide(NewService),
	kessoku.Bind[Logger](kessoku.Provide(NewConsoleLogger)),
	DBSet,
)
