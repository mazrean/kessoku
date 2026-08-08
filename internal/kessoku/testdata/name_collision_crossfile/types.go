package main

type Foo struct{}

// NewFoo already exists in another file — collision with injector name "NewFoo"
func NewFoo() *Foo { return &Foo{} }

func NewFooImpl() *Foo { return &Foo{} }

func main() {}
