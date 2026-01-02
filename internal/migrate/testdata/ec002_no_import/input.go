package ec002_no_import

// This file does not import wire at all
// The migration should skip this file with a warning

type Foo struct{}

func NewFoo() *Foo { return &Foo{} }
