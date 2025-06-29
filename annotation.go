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

type iFnProvider[T any] interface {
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
//
// Example:
//
//	kessoku.Provide(NewDatabase)  // where NewDatabase returns (*Database, error)
//	kessoku.Provide(NewService)   // where NewService returns *Service
func Provide[T any](fn T) fnProvider[T] {
	return fnProvider[T]{fn: fn}
}

// asProvider represents a type binding that maps one type to another.
// S is the source type and T is the target type that the binding maps to.
type asProvider[Dst, Fn any, Provider iFnProvider[Fn]] struct {
	provider Provider
}

// provide implements the provider interface for asProvider.
func (p asProvider[_, _, _]) provide() {}

// Fn returns the wrapped function for the as provider.
// This method is used internally by the code generator.
func (p asProvider[_, Fn, _]) Fn() Fn {
	return p.provider.Fn()
}

// As creates a type binding that maps type S to type T using the given provider.
// This is useful when you want to provide a concrete implementation for an interface
// or when you need to map one type to another in the dependency graph.
//
// Example:
//
//	kessoku.As[UserRepository](kessoku.Provide(NewDatabaseUserRepo))
//
// This tells kessoku that when a UserRepository is needed, it should use
// the DatabaseUserRepo implementation provided by NewDatabaseUserRepo.
func As[Dst, Fn any, Provider iFnProvider[Fn]](fn Provider) asProvider[Dst, Fn, Provider] {
	return asProvider[Dst, Fn, Provider]{provider: fn}
}

type asMapProvider[Dst, Src, Fn any, Provider iFnProvider[Fn]] struct {
	provider Provider
}

func (p asMapProvider[_, _, _, _]) provide() {}

func (p asMapProvider[_, _, Fn, _]) Fn() Fn {
	return p.provider.Fn()
}

// AsMap creates a type binding that maps a source type Src to a destination type Dst.
// This is typically used to map concrete types to interface types in a more explicit way,
// where you want to specify both the source and destination types clearly.
//
// Example:
//
//	kessoku.AsMap[UserRepository, *DatabaseUserRepo](kessoku.Provide(NewDatabaseUserRepo))
//
// This tells kessoku that when a UserRepository (Dst) is needed, it should use
// the *DatabaseUserRepo (Src) implementation provided by NewDatabaseUserRepo.
// The source type Src is explicitly declared for better type safety and clarity.
func AsMap[Dst, Src, Fn any, Provider iFnProvider[Fn]](fn Provider) asMapProvider[Dst, Src, Fn, Provider] {
	return asMapProvider[Dst, Src, Fn, Provider]{provider: fn}
}

// Value creates a provider for a constant value.
// This is useful when you want to inject configuration values, constants,
// or other static data into your dependency graph.
//
// Example:
//
//	kessoku.Value("localhost:8080")  // provides a string value
//	kessoku.Value(42)               // provides an int value
//	kessoku.Value(&Config{...})     // provides a config struct
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
// Example:
//
//	var _ = kessoku.Inject[*App](
//		"InitializeApp",
//		kessoku.Provide(NewConfig),
//		kessoku.Provide(NewDatabase),
//		kessoku.Provide(NewUserService),
//		kessoku.Provide(NewApp),
//	)
//
// If NewConfig requires a string parameter that's not provided,
// this generates a function like:
//
//	func InitializeApp(arg0 string) (*App, error) {
//		config := NewConfig(arg0)
//		db, err := NewDatabase(config)
//		if err != nil { return nil, err }
//		userService := NewUserService(db)
//		app := NewApp(userService)
//		return app, nil
//	}
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

func Set(providers ...provider) set {
	return set{}
}
