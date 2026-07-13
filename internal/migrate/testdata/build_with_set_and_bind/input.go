//go:build wireinject

package build_with_set_and_bind

import "github.com/google/wire"

type Iface interface{ Do() }

type Impl struct{}

func (i *Impl) Do() {}

func NewImpl() *Impl { return &Impl{} }

type App struct{ i Iface }

func NewApp(i Iface) *App { return &App{i: i} }

// MySet contains both the constructor and the wire.Bind for the interface.
// When wire.Build references MySet, the set must not suppress itself
// from the Inject call due to its own internal Bind (QA-18).
var MySet = wire.NewSet(NewImpl, wire.Bind(new(Iface), new(*Impl)))

func InitializeApp() *App {
	wire.Build(MySet, NewApp)
	return nil
}
