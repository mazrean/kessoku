//go:build wireinject

package fieldsof_external

import (
	"github.com/google/wire"
	"github.com/mazrean/kessoku/internal/migrate/testdata/fieldsof_external/external"
)

var ConfigSet = wire.NewSet(wire.FieldsOf(new(*external.Config), "DB", "Cache"))
