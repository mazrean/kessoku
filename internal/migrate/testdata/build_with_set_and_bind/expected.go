//go:generate go tool kessoku $GOFILE

package build_with_set_and_bind

import (
	"github.com/mazrean/kessoku"
)

var MySet = kessoku.Set(
	kessoku.Bind[Iface](kessoku.Provide(NewImpl)),
)
var _ = kessoku.Inject[*App](
	"InitializeApp",
	MySet,
	kessoku.Provide(NewApp),
)
