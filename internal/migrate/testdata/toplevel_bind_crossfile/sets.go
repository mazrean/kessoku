//go:build wireinject

package toplevel_bind_crossfile

import "github.com/google/wire"

type App struct {
	svc Svc
}

func NewApp(svc Svc) *App {
	return &App{svc: svc}
}

// AppSet references SvcBind which is defined in binds.go (cross-file bind variable).
var AppSet = wire.NewSet(NewSvcImpl, SvcBind)
