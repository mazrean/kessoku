//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	. "github.com/mazrean/kessoku/internal/kessoku/testdata/dot_import_set/setpkg"
)

// InitializeService initializes the service using a dot-imported Set variable.
var _ = kessoku.Inject[*Service](
	"InitializeService",
	ConfigSet,
	kessoku.Provide(NewService),
)
