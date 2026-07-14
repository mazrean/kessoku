//go:build wireinject

package bind_referenced_set

import "github.com/google/wire"

type MyInterface interface {
	Do() string
}

type MyImpl struct{}

func (m *MyImpl) Do() string { return "impl" }

func NewMyImpl() *MyImpl { return &MyImpl{} }

var ImplSet = wire.NewSet(NewMyImpl)

var Set = wire.NewSet(ImplSet, wire.Bind(new(MyInterface), new(*MyImpl)))
