//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package bind_with_struct

import (
	"github.com/mazrean/kessoku"
)

var MySet = kessoku.Set(
	kessoku.Provide(func(name string) MyStruct {
		return MyStruct{Name: name}
	}),
	kessoku.Provide(func(name string) *MyStruct {
		return &MyStruct{Name: name}
	}),
	kessoku.Bind[Stringer](kessoku.Provide(func(name string) *MyStruct {
		return &MyStruct{Name: name}
	})),
)
