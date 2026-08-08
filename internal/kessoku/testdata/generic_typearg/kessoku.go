//go:generate go tool kessoku $GOFILE

package main

import (
	"net/http"
	"sync/atomic"

	"github.com/mazrean/kessoku"
)

// Test that imports for generic type arguments are correctly included in
// generated band files. The injector return type is *atomic.Pointer[*http.Client],
// which requires both "sync/atomic" (outer type) and "net/http" (type argument)
// to be imported in the generated file.
var _ = kessoku.Inject[*atomic.Pointer[*http.Client]](
	"InitializeHTTPClientPtr",
	kessoku.Provide(NewHTTPClientPtr),
)
