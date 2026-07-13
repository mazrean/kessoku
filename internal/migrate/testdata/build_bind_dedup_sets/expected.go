//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package build_bind_dedup_sets

import (
	"github.com/mazrean/kessoku"
)

var ImplSet = kessoku.Set(
	kessoku.Provide(NewSvcImpl),
)
var BindSet = kessoku.Set(
	kessoku.Bind[Svc](kessoku.Provide(NewSvcImpl)),
)
var _ = kessoku.Inject[*App](
	"InitializeApp",
	BindSet,
	kessoku.Provide(NewApp),
)
