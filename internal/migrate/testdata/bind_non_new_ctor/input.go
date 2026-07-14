//go:build wireinject

package bind_non_new_ctor

import "github.com/google/wire"

type MyInterface interface {
	Do() string
}

type MyImpl struct{}

func (m *MyImpl) Do() string { return "impl" }

func MakeMyImpl() *MyImpl { return &MyImpl{} }

var Set = wire.NewSet(
	MakeMyImpl,
	wire.Bind(new(MyInterface), new(*MyImpl)),
)
