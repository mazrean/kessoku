//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package struct_keyword_field

import (
	"github.com/mazrean/kessoku"
)

var Set = kessoku.Set(
	kessoku.Provide(func(type_ string, name string) EventLog {
		return EventLog{Type: type_, Name: name}
	}),
	kessoku.Provide(func(type_ string, name string) *EventLog {
		return &EventLog{Type: type_, Name: name}
	}),
)
