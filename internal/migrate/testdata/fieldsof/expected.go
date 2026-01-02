//go:generate go tool kessoku $GOFILE

package fieldsof

import (
	"github.com/mazrean/kessoku"
)

var FieldsSet = kessoku.Set(
	kessoku.Provide(func(s *Config) (*DBConn, *CacheConn) {
		return s.DB, s.Cache
	}),
)
