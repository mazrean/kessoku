//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// RouteMap holds URL routes.
type RouteMap map[string]string

// NewRoutes creates a route mapping.
func NewRoutes() RouteMap {
	return RouteMap{
		"/api/users":    "users",
		"/api/products": "products",
	}
}

// HandlerMap maps route names to handlers.
type HandlerMap map[string]func()

// NewHandlers creates handler mappings.
func NewHandlers() HandlerMap {
	return HandlerMap{
		"users":    func() {},
		"products": func() {},
	}
}

// Router uses route and handler maps.
type Router struct {
	routes   RouteMap
	handlers HandlerMap
}

// NewRouter creates a new router.
func NewRouter(routes RouteMap, handlers HandlerMap) *Router {
	return &Router{routes: routes, handlers: handlers}
}

// Test map types as dependencies
var _ = kessoku.Inject[*Router](
	"InitializeRouter",
	kessoku.Provide(NewRoutes),
	kessoku.Provide(NewHandlers),
	kessoku.Provide(NewRouter),
)
