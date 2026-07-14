package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// This file declares both the injector name and the function with the same name
// to verify that kessoku detects the collision and fails with a clear error message.
var _ = kessoku.Inject[*Foo](
	"NewFoo",
	kessoku.Provide(NewFooImpl),
)

type Foo struct{}

// NewFoo already exists — collision with injector name "NewFoo"
func NewFoo() *Foo { return &Foo{} }

func NewFooImpl() *Foo { return &Foo{} }

func main() {}
