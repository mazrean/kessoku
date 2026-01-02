package basic

import "github.com/google/wire"

var SuperSet = wire.NewSet(NewFoo)

type Foo struct{}

func NewFoo() *Foo {
	return &Foo{}
}
