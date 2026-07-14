//go:build wireinject

package bind_with_struct

import "github.com/google/wire"

type Stringer interface {
	String() string
}

type MyStruct struct {
	Name string
}

func (m *MyStruct) String() string {
	return m.Name
}

var MySet = wire.NewSet(
	wire.Struct(new(MyStruct), "Name"),
	wire.Bind(new(Stringer), new(*MyStruct)),
)
