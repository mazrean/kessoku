//go:generate go tool kessoku $GOFILE

package ec005_nested_newset

import (
	"github.com/mazrean/kessoku"
)

var FooSet = kessoku.Set(
	kessoku.Provide(NewFoo),
)
var AllSet = kessoku.Set(
	FooSet,
	kessoku.Provide(NewBar),
)
