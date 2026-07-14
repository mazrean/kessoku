//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// App uses a func() as a dependency.
type App struct {
	handler func()
}

// NewApp creates an App with a handler function.
func NewApp(h func()) *App {
	return &App{handler: h}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Value(func() { println("hello") }),
	kessoku.Provide(NewApp),
)
