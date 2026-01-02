package importreplace

import (
	"fmt"

	"github.com/google/wire"
)

var MySet = wire.NewSet(NewPrinter)

func NewPrinter() *Printer {
	return &Printer{}
}

type Printer struct{}

func (p *Printer) Print(s string) {
	fmt.Println(s)
}
