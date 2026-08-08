//go:build wireinject

package build_bind_dedup_sets

import "github.com/google/wire"

type Svc interface {
	Do() string
}

type SvcImpl struct{}

func (s *SvcImpl) Do() string { return "done" }

func NewSvcImpl() *SvcImpl { return &SvcImpl{} }

type App struct{ svc Svc }

func NewApp(s Svc) *App { return &App{svc: s} }

// ImplSet provides the concrete type.
var ImplSet = wire.NewSet(NewSvcImpl)

// BindSet binds the interface to the concrete type.
var BindSet = wire.NewSet(wire.Bind(new(Svc), new(*SvcImpl)))

func InitializeApp() *App {
	wire.Build(ImplSet, BindSet, NewApp)
	return nil
}
