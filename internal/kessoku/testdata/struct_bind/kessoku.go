//go:generate go tool kessoku $GOFILE

// Package main demonstrates Bind[I](Struct[T]()) which should bind
// the struct as an interface without leaking the interface as an external parameter.
package main

import "github.com/mazrean/kessoku"

// InitializeDatabase is the generated injector function.
// ConfigProvider should be resolved from *Config via Struct expansion + Bind,
// not leaked as an external parameter.
var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	kessoku.Provide(NewConfig),
	kessoku.Bind[ConfigProvider](kessoku.Struct[*Config]()),
	kessoku.Provide(NewDatabase),
)
