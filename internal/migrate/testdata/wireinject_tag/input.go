//go:build wireinject

package wireinjecttag

import "github.com/google/wire"

var TestSet = wire.NewSet(NewFoo)

func NewFoo() *Foo { return &Foo{} }

type Foo struct{}
