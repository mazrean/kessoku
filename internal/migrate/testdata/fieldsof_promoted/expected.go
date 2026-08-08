//go:generate go tool kessoku $GOFILE

package fieldsof_promoted

import (
	"github.com/mazrean/kessoku"
)

var FieldsSet = kessoku.Set(
	kessoku.Provide(func(s Config) *string {
		return s.DB
	}),
)
