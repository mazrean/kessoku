//go:build wireinject

package toplevel_bind_crossfile

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

// SvcBind is defined in this file; AppSet is in sets.go.
var SvcBind = wire.Bind(new(Svc), new(*SvcImpl))
