package merge

import (
	"github.com/google/wire"
)

type Bar struct{}

func NewBar() *Bar { return &Bar{} }

var BarSet = wire.NewSet(NewBar)
