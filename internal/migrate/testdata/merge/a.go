package merge

import (
	"github.com/google/wire"
)

type Foo struct{}

func NewFoo() *Foo { return &Foo{} }

var FooSet = wire.NewSet(NewFoo)
