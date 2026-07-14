//go:generate go tool kessoku $GOFILE

package dual_bind_same_impl

import (
	"github.com/mazrean/kessoku"
)

var RWSet = kessoku.Set(
	kessoku.Bind[Writer](kessoku.Bind[Reader](kessoku.Provide(NewRWImpl))),
)
