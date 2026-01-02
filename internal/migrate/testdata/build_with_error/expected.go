package build_with_error

import (
	"github.com/mazrean/kessoku"
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewDB),
	kessoku.Provide(NewApp),
)
