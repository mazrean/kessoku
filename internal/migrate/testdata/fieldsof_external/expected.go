package fieldsof_external

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/migrate/testdata/fieldsof_external/external"
)

var ConfigSet = kessoku.Set(
	kessoku.Provide(func(s *external.Config) (string, string) {
		return s.DB, s.Cache
	}),
)
