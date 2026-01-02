package ec001_no_patterns

import (
	"github.com/google/wire"
)

// This file imports wire but doesn't use any wire patterns
// The migration should skip this file with a warning

var _ = wire.Bind // Just to satisfy import, not a wire pattern

type Foo struct{}

func NewFoo() *Foo { return &Foo{} }
