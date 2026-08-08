//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// NewEventHandler provides a bare func() as an injectable value.
// This must NOT be confused with a wire-style cleanup function.
func NewEventHandler() func() {
	return func() { println("hello") }
}

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
	kessoku.Provide(NewEventHandler),
	kessoku.Provide(NewApp),
)
