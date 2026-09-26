//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_async_bind/pkgx"
)

// App depends on the bound interface.
type App struct {
	G pkgx.Greeter
	O *pkgx.Other
}

// NewApp creates a new App.
func NewApp(g pkgx.Greeter, o *pkgx.Other) *App {
	return &App{G: g, O: o}
}

// InitializeApp uses an async Bind whose implementation is unexported.
var _ = kessoku.Inject[*App](
	"InitializeApp",
	pkgx.Set,
	kessoku.Provide(NewApp),
)

func main() {}
