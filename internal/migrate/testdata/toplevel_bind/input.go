package toplevel_bind

import "github.com/google/wire"

type Svc interface {
	Do() string
}

type SvcImpl struct{}

func (s *SvcImpl) Do() string {
	return "svc"
}

func NewSvcImpl() *SvcImpl {
	return &SvcImpl{}
}

type App struct {
	svc Svc
}

func NewApp(svc Svc) *App {
	return &App{svc: svc}
}

var SvcBind = wire.Bind(new(Svc), new(*SvcImpl))

var AppSet = wire.NewSet(NewSvcImpl, SvcBind)
