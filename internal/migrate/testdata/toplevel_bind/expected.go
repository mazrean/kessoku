//go:generate go tool kessoku $GOFILE

package toplevel_bind

import (
	"github.com/mazrean/kessoku"
)

var SvcBind = kessoku.Set(
	kessoku.Bind[Svc](kessoku.Provide(NewSvcImpl)),
)
var AppSet = kessoku.Set(
	SvcBind,
)
