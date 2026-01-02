package pkg

type Foo struct {
	Bar *Bar
}

type Bar struct{}

func NewFoo(bar *Bar) *Foo {
	return &Foo{Bar: bar}
}

func NewBar() *Bar {
	return &Bar{}
}
