package dual_bind_same_impl

import "github.com/google/wire"

type Reader interface {
	Read() string
}

type Writer interface {
	Write(s string)
}

type RWImpl struct{}

func (r *RWImpl) Read() string  { return "read" }
func (r *RWImpl) Write(s string) {}

func NewRWImpl() *RWImpl {
	return &RWImpl{}
}

// RWSet binds two interfaces to the same concrete type.
var RWSet = wire.NewSet(
	NewRWImpl,
	wire.Bind(new(Reader), new(*RWImpl)),
	wire.Bind(new(Writer), new(*RWImpl)),
)
