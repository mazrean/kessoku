package bind_with_inline_nested_set

import "github.com/google/wire"

type Iface interface {
	Do() string
}

type Impl struct{}

func (i *Impl) Do() string {
	return "impl"
}

func NewImpl() *Impl {
	return &Impl{}
}

// wire.Bind is a sibling of an inline wire.NewSet containing the provider.
// The provider (NewImpl) should be filtered out from the nested set because
// it is bound via wire.Bind in the outer scope.
var AllSet = wire.NewSet(
	wire.NewSet(NewImpl),
	wire.Bind(new(Iface), new(*Impl)),
)
