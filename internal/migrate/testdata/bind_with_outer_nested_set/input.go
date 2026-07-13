package bind_with_outer_nested_set

import "github.com/google/wire"

type IfaceC interface{ C() }
type ImplC struct{}

func (c *ImplC) C() {}

func NewImplC() *ImplC { return &ImplC{} }

var ImplCSet = wire.NewSet(NewImplC)
var OuterSet = wire.NewSet(wire.NewSet(ImplCSet))
var BindSet = wire.NewSet(wire.Bind(new(IfaceC), new(*ImplC)))
var UseSet = wire.NewSet(OuterSet, BindSet)
