//go:build wireinject

package dual_named_sets_same_impl

import "github.com/google/wire"

type IfaceA interface {
	DoA() string
}

type IfaceB interface {
	DoB() string
}

type Impl struct{}

func (i *Impl) DoA() string { return "a" }
func (i *Impl) DoB() string { return "b" }

func NewImpl() *Impl { return &Impl{} }

type App struct{}

func NewApp(a IfaceA, b IfaceB) *App { return &App{} }

// SetA and SetB each bind a different interface to the same concrete *Impl.
// When both sets are combined, wire internally shares the single NewImpl provider.
// The migrated output must not produce duplicate kessoku.Provide(NewImpl) entries.
var SetA = wire.NewSet(NewImpl, wire.Bind(new(IfaceA), new(*Impl)))
var SetB = wire.NewSet(NewImpl, wire.Bind(new(IfaceB), new(*Impl)))

var BigSet = wire.NewSet(SetA, SetB)

func InitApp() *App {
	wire.Build(BigSet, NewApp)
	return nil
}
