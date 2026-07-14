//go:generate go tool kessoku $GOFILE

package fieldsof_external

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/migrate/testdata/fieldsof_external/external"
)

var ConfigSet = kessoku.Set(
	kessoku.Provide(func(s *external.Config) (string, *string, string, *string) {
		return s.DB, &s.DB, s.Cache, &s.Cache
	}),
)
