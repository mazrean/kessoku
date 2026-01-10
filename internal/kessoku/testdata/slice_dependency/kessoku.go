//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Middleware represents a middleware function.
type Middleware struct {
	Name string
}

// MiddlewareList is a slice of middlewares.
type MiddlewareList []*Middleware

// NewMiddlewares creates a list of middlewares.
func NewMiddlewares() MiddlewareList {
	return MiddlewareList{
		{Name: "auth"},
		{Name: "logging"},
		{Name: "cors"},
	}
}

// Handler represents an HTTP handler.
type Handler struct {
	middlewares MiddlewareList
}

// NewHandler creates a new handler with middlewares.
func NewHandler(middlewares MiddlewareList) *Handler {
	return &Handler{middlewares: middlewares}
}

// Test slice type as dependency
var _ = kessoku.Inject[*Handler](
	"InitializeHandler",
	kessoku.Provide(NewMiddlewares),
	kessoku.Provide(NewHandler),
)
