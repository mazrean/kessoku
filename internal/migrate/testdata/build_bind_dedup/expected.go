//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package build_bind_dedup

import (
	"github.com/mazrean/kessoku"
)

var GreeterSet = kessoku.Set(
	kessoku.Provide(NewSpanishGreeter),
)
var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Bind[Greeter](kessoku.Provide(NewSpanishGreeter)),
	kessoku.Provide(NewApp),
)
