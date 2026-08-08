//go:generate go tool kessoku $GOFILE

package bind_with_outer_nested_set

import (
	"github.com/mazrean/kessoku"
)

var ImplCSet = kessoku.Set(
	kessoku.Provide(NewImplC),
)
var OuterSet = kessoku.Set(
	ImplCSet,
)
var BindSet = kessoku.Set(
	kessoku.Bind[IfaceC](kessoku.Provide(NewImplC)),
)
var UseSet = kessoku.Set(
	BindSet,
)
