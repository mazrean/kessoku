//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package bind_non_new_ctor

import (
	"github.com/mazrean/kessoku"
)

var Set = kessoku.Set(
	kessoku.Bind[MyInterface](kessoku.Provide(MakeMyImpl)),
)
