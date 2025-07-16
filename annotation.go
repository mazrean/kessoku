// Package kessoku provides annotation-based dependency injection code generation for Go.
package kessoku

// name represents an identifier for injectors and arguments.
// It's used to specify the name of generated injector functions
// and to identify function parameters in dependency injection.
type name string

// provider is a marker interface that all dependency providers must implement.
// This interface is used internally by the code generator to identify
// different types of providers in the Inject function calls.
type provider interface {
	provide()
}

type funcProvider[T any] interface {
	provider
	Fn() T
}

// fnProvider wraps a function to be used as a dependency provider.
// The generic type T should be a function type that returns one or more values,
// where the returned values become available dependencies.
type fnProvider[T any] struct {
	fn T
}

// provide implements the provider interface for fnProvider.
func (p fnProvider[T]) provide() {}

// Fn returns the wrapped function.
// This method is used internally by the code generator.
func (p fnProvider[T]) Fn() T {
	return p.fn
}

// Provide wraps a function to be used as a dependency provider.
// The function fn should return one or more values that can be injected
// as dependencies into other functions.
func Provide[T any](fn T) fnProvider[T] {
	return fnProvider[T]{fn: fn}
}

type asyncProvider[T any, F funcProvider[T]] struct {
	fn F
}

func (p asyncProvider[T, F]) provide() {}

func (p asyncProvider[T, F]) Fn() T {
	return p.fn.Fn()
}

// Async wraps a provider to enable parallel execution with other async providers.
// Async providers that don't depend on each other will be executed concurrently,
// improving performance for slow operations like database connections or API calls.
// When any async provider is present, the generated injector function will include
// a context.Context parameter for cancellation and timeout support.
func Async[T any, F funcProvider[T]](fn F) asyncProvider[T, F] {
	return asyncProvider[T, F]{fn: fn}
}

// bindProvider represents a type binding that maps one type to another.
// S is the source type and T is the target type that the binding maps to.
type bindProvider[S, T any, F funcProvider[T]] struct {
	fn F
}

// provide implements the provider interface for bindProvider.
func (p bindProvider[_, _, F]) provide() {}

// Fn returns the wrapped function for the bind provider.
// This method is used internally by the code generator.
func (p bindProvider[_, T, _]) Fn() T {
	return p.fn.Fn()
}

// Bind creates a type binding that maps type S to type T using the given provider.
// This is useful when you want to provide a concrete implementation for an interface
// or when you need to map one type to another in the dependency graph.
func Bind[S, T any, F funcProvider[T]](fn F) bindProvider[S, T, F] {
	return bindProvider[S, T, F]{fn: fn}
}

// Value creates a provider for a constant value.
// This is useful when you want to inject configuration values, constants,
// or other static data into your dependency graph.
func Value[T any](v T) fnProvider[func() T] {
	return fnProvider[func() T]{
		fn: func() T { return v },
	}
}

// Inject declares a dependency injection build directive.
// It generates a function with the specified name that constructs and returns
// an instance of type T using the provided dependency providers.
//
// The generated function will:
// 1. Automatically detect missing dependencies and add them as function parameters
// 2. Call provider functions in the correct order based on dependencies
// 3. Return an instance of type T (and error if any provider returns an error)
//
// Use go:generate to trigger code generation:
//
//	//go:generate go tool kessoku $GOFILE
func Inject[T any](name name, providers ...provider) struct{} {
	// This function is analyzed at compile time by the kessoku code generator.
	// The actual implementation is generated and written to *_band.go files.
	return struct{}{}
}

type set struct{}

func (s set) provide() {}

// Set groups multiple providers together as a reusable unit.
// This is useful for organizing related providers that are commonly used together,
// such as database-related providers or authentication-related providers.
func Set(providers ...provider) set {
	return set{}
}
