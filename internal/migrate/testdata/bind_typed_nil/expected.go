//go:generate go tool kessoku $GOFILE

package bind_typed_nil

import (
	"github.com/mazrean/kessoku"
)

var RepoSet = kessoku.Set(
	kessoku.Provide(NewPostgresRepo),
)
