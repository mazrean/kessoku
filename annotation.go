// Package kessoku provides fast, parallel dependency injection code generation for Go.
// 
// Kessoku makes your Go applications start 2.25x faster by executing independent
// dependencies in parallel instead of sequentially. Perfect for speeding up
// database connections, API initializations, and other slow startup operations.
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
//
// Use this for any function that creates dependencies like databases, services, or configs.
// The function will be called during dependency injection to provide its return values.
//
// Example:
//	kessoku.Provide(NewDatabase)  // func NewDatabase() (*sql.DB, error)
//	kessoku.Provide(NewLogger)    // func NewLogger() *log.Logger
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

// Async enables parallel execution for slow providers, improving startup performance.
//
// Wrap slow operations (DB connections, API calls, cache setup) with Async() to run them
// in parallel instead of waiting for each one sequentially. Perfect for cutting startup time.
//
// Example: 450ms → 200ms faster startup
//	kessoku.Async(kessoku.Provide(NewDatabase))    // 200ms }
//	kessoku.Async(kessoku.Provide(NewCache))       // 150ms } All run in parallel!
//	kessoku.Async(kessoku.Provide(NewAPIClient))   // 100ms }
//
// The generated injector will automatically include context.Context for cancellation.
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

// Bind creates an interface-to-implementation mapping for cleaner dependency injection.
//
// Use this when you want to inject interfaces but provide concrete implementations.
// Perfect for testing (swap real implementations with mocks) and clean architecture.
//
// Example:
//	kessoku.Bind[UserRepository](kessoku.Provide(NewPostgresUserRepo))
//	// Now anywhere UserRepository is needed, PostgresUserRepo will be injected
func Bind[S, T any, F funcProvider[T]](fn F) bindProvider[S, T, F] {
	return bindProvider[S, T, F]{fn: fn}
}

// Value injects constant values like config settings, feature flags, or static data.
//
// Use this for any constant that your services need - no function creation required!
// Perfect for environment variables, API keys, timeouts, and configuration values.
//
// Example:
//	kessoku.Value("database-url"),     // Inject string constant
//	kessoku.Value(30*time.Second),     // Inject timeout duration
//	kessoku.Value(map[string]string{   // Inject config map
//	    "env": "production",
//	})
func Value[T any](v T) fnProvider[func() T] {
	return fnProvider[func() T]{
		fn: func() T { return v },
	}
}

// Inject creates a dependency injection build directive that generates fast initialization code.
//
// Declare your dependencies once, get blazing-fast startup automatically! Kessoku analyzes
// your providers and generates optimized code that runs independent operations in parallel.
//
// Example - creates InitializeApp() function:
//	var _ = kessoku.Inject[*App](
//	    "InitializeApp",                                   // Generated function name
//	    kessoku.Async(kessoku.Provide(NewDatabase)),      // Runs in parallel
//	    kessoku.Async(kessoku.Provide(NewCache)),         // Runs in parallel
//	    kessoku.Provide(NewApp),                          // Waits for dependencies
//	)
//
// Generated function automatically:
// • Runs async providers in parallel for maximum speed
// • Handles dependency order and error propagation  
// • Includes context.Context for cancellation when async providers are used
//
// Trigger code generation:
//	//go:generate go tool kessoku $GOFILE
func Inject[T any](name name, providers ...provider) struct{} {
	// This function is analyzed at compile time by the kessoku code generator.
	// The actual implementation is generated and written to *_band.go files.
	return struct{}{}
}

type set struct{}

func (s set) provide() {}

// Set groups related providers into reusable bundles for better organization.
//
// Perfect for organizing providers by domain (database, auth, monitoring) and reusing
// them across multiple injectors. Keeps your dependency declarations clean and DRY.
//
// Example:
//	var DatabaseSet = kessoku.Set(
//	    kessoku.Provide(NewConfig),
//	    kessoku.Async(kessoku.Provide(NewDatabase)),
//	    kessoku.Provide(NewMigrator),
//	)
//	
//	// Reuse the set in multiple injectors
//	var _ = kessoku.Inject[*App]("InitApp", DatabaseSet, kessoku.Provide(NewApp))
func Set(providers ...provider) set {
	return set{}
}
