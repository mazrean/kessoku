//go:build wireinject

package build_bind_dedup

import "github.com/google/wire"

type Greeter interface {
	Greet() string
}

type SpanishGreeter struct{}

func (g *SpanishGreeter) Greet() string {
	return "Hola"
}

func NewSpanishGreeter() *SpanishGreeter {
	return &SpanishGreeter{}
}

type App struct {
	greeter Greeter
}

func NewApp(g Greeter) *App {
	return &App{greeter: g}
}

var GreeterSet = wire.NewSet(NewSpanishGreeter)

func InitializeApp() *App {
	wire.Build(GreeterSet, wire.Bind(new(Greeter), new(*SpanishGreeter)), NewApp)
	return nil
}
