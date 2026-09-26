package main

import "github.com/mazrean/kessoku/internal/kessoku/testdata/external_set/pkga"

// App is the application root.
type App struct {
	a *pkga.A
}

// NewApp creates a new App.
func NewApp(a *pkga.A) *App {
	return &App{a: a}
}

func main() {}
