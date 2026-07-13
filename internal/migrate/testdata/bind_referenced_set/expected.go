//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package bind_referenced_set

import (
	"github.com/mazrean/kessoku"
)

var ImplSet = kessoku.Set(
	kessoku.Provide(NewMyImpl),
)
var Set = kessoku.Set(
	kessoku.Bind[MyInterface](kessoku.Provide(NewMyImpl)),
)
