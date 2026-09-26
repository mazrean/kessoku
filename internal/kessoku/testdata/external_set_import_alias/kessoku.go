//go:generate go tool kessoku $GOFILE

package main

import (
	"github.com/mazrean/kessoku"
	"github.com/mazrean/kessoku/internal/kessoku/testdata/external_set_import_alias/setpkg"
)

// InitializeWrapper uses a Set whose package imports depimpl under an alias.
var _ = kessoku.Inject[*setpkg.Wrapper](
	"InitializeWrapper",
	setpkg.Set,
)

func main() {}
