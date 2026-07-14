//go:generate go tool kessoku $GOFILE

package dual_named_sets_same_impl

import (
	"github.com/mazrean/kessoku"
)

var SetA = kessoku.Set(
	kessoku.Bind[IfaceA](kessoku.Provide(NewImpl)),
)
var SetB = kessoku.Set(
	kessoku.Bind[IfaceB](kessoku.Provide(NewImpl)),
)
var BigSet = kessoku.Set(
	kessoku.Bind[IfaceB](kessoku.Bind[IfaceA](kessoku.Provide(NewImpl))),
)
var _ = kessoku.Inject[*App](
	"InitApp",
	BigSet,
	kessoku.Provide(NewApp),
)
