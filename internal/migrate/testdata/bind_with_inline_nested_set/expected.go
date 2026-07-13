//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package bind_with_inline_nested_set

import (
	"github.com/mazrean/kessoku"
)

var AllSet = kessoku.Set(
	kessoku.Bind[Iface](kessoku.Provide(NewImpl)),
)
