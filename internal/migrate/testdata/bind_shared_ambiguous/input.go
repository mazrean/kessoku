//go:build wireinject

package bind_shared_ambiguous

import "github.com/google/wire"

type Iface interface {
	Do() string
}

type Impl struct{}

func (s *Impl) Do() string { return "impl" }

// Two constructors that both return *Impl — migration cannot pick one automatically.
func NewImplV1() *Impl { return &Impl{} }
func NewImplV2() *Impl { return &Impl{} }

type App struct{}

func NewApp(i Iface) *App { return &App{} }

// BindIface is a shared bind set referenced by both injectors.
var BindIface = wire.NewSet(wire.Bind(new(Iface), new(*Impl)))

func InitializeV1() *App {
	wire.Build(BindIface, NewImplV1, NewApp)
	return nil
}

func InitializeV2() *App {
	wire.Build(BindIface, NewImplV2, NewApp)
	return nil
}
