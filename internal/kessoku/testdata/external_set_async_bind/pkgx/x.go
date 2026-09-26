package pkgx

import "github.com/mazrean/kessoku"

// Greeter is the exported interface.
type Greeter interface {
	Greet() string
}

type greeterImpl struct{}

func (greeterImpl) Greet() string { return "hi" }

// NewGreeter returns an unexported implementation.
func NewGreeter() (*greeterImpl, error) { return &greeterImpl{}, nil }

// Other is an independent async dependency.
type Other struct{}

// NewOther creates a new Other.
func NewOther() (*Other, error) { return &Other{}, nil }

// Set hides the implementation behind Greeter.
var Set = kessoku.Set(
	kessoku.Async(kessoku.Bind[Greeter](kessoku.Provide(NewGreeter))),
	kessoku.Async(kessoku.Provide(NewOther)),
)
