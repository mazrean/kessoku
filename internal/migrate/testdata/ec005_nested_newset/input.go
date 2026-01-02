package ec005_nested_newset

import (
	"github.com/google/wire"
)

type Foo struct{}
type Bar struct{}

func NewFoo() *Foo { return &Foo{} }
func NewBar() *Bar { return &Bar{} }

var FooSet = wire.NewSet(NewFoo)
var AllSet = wire.NewSet(FooSet, NewBar)
