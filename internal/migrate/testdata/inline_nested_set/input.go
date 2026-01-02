package inline_nested_set

import (
	"github.com/google/wire"
)

type Foo struct{}
type Bar struct{}

func NewFoo() *Foo { return &Foo{} }
func NewBar() *Bar { return &Bar{} }

// Inline nested NewSet within a NewSet
var AllSet = wire.NewSet(
	wire.NewSet(NewFoo),
	NewBar,
)
