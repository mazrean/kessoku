# Kessoku

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)

A dependency injection code generator for Go, similar to [google/wire](https://github.com/google/wire). Kessoku generates Go code for compile-time dependency injection, eliminating runtime reflection overhead.

## Features

Kessoku extends the concept of compile-time dependency injection with advanced features that set it apart from google/wire:

### 🚀 **Advanced Parallel Processing**
- **Async Provider Support**: Execute independent providers concurrently using `kessoku.Async()`
- **Intelligent Dependency Ordering**: Maintains correct execution order while maximizing parallelism
- **Channel-based Synchronization**: Advanced coordination between dependent async providers
- **Context Integration**: Automatic `context.Context` injection for timeout and cancellation support

### ⚡ **Compile-time Optimization**
- **Zero Runtime Overhead**: All dependency resolution happens at compile time
- **Static Code Generation**: Generates optimized Go code with no reflection
- **Type Safety**: Full compile-time type checking and validation
- **Performance**: Minimal memory allocations and optimal execution paths

### 🔧 **Enhanced Developer Experience**
- **Automatic Context Injection**: No manual context passing for async operations
- **Comprehensive Error Handling**: Proper error propagation across async boundaries
- **Go Generate Integration**: Seamless workflow with `go generate`
- **Cycle Detection**: Prevents circular dependencies at compile time

## Installation

### Recommended: Go Tool

```bash
go get -tool github.com/mazrean/kessoku/cmd/kessoku@latest
```

This installs kessoku as a Go tool, making it available via `go tool kessoku`.

### Alternative: Direct Install

```bash
go install github.com/mazrean/kessoku/cmd/kessoku@latest
```

### From Releases

Download the latest binary from the [releases page](https://github.com/mazrean/kessoku/releases).

### Via Homebrew

```bash
brew install mazrean/tap/kessoku
```

## Quick Start

### 1. Create Provider Functions

```go
package main

import "github.com/mazrean/kessoku"

type Config struct {
    DatabaseURL string
    Port        int
}

func NewConfig() *Config {
    return &Config{
        DatabaseURL: "postgres://localhost/mydb",
        Port:        8080,
    }
}

type Database struct {
    url string
}

func NewDatabase(config *Config) (*Database, error) {
    return &Database{url: config.DatabaseURL}, nil
}

type UserService struct {
    db *Database
}

func NewUserService(db *Database) *UserService {
    return &UserService{db: db}
}

type App struct {
    config  *Config
    service *UserService
}

func NewApp(config *Config, service *UserService) *App {
    return &App{
        config:  config,
        service: service,
    }
}
```

### 2. Define Injector Function

Create a file `kessoku.go` with injector declarations:

```go
package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// InitializeApp creates the application with all dependencies.
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDatabase),
    kessoku.Provide(NewUserService),
    kessoku.Provide(NewApp),
)
```

### 3. Generate Dependency Injection Code

```bash
# Using go generate (recommended)
go generate ./...

# Or run kessoku directly
go tool kessoku kessoku.go
```

This generates `*_band.go` files with the actual dependency injection implementation.

### 4. Use the Generated Code

```go
func main() {
    app, err := InitializeApp()
    if err != nil {
        log.Fatal("Failed to initialize app:", err)
    }
    
    // Use your app
    fmt.Printf("App running on port %d\n", app.config.Port)
}
```

## Async Provider Support

Kessoku's key differentiator is its support for async providers that execute in parallel while maintaining dependency order:

### Async Provider Example

```go
// Async providers for slow operations
func NewDatabaseService() (*DatabaseService, error) {
    // Simulate slow database connection
    time.Sleep(200 * time.Millisecond)
    return &DatabaseService{}, nil
}

func NewCacheService() *CacheService {
    // Simulate slow cache connection
    time.Sleep(150 * time.Millisecond) 
    return &CacheService{}
}

func NewMessagingService() *MessagingService {
    // Simulate slow messaging setup
    time.Sleep(180 * time.Millisecond)
    return &MessagingService{}
}
```

### Async Injector Declaration

```go
//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Async providers execute in parallel
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDatabaseService)),  // Parallel execution
    kessoku.Async(kessoku.Provide(NewCacheService)),     // Parallel execution
    kessoku.Async(kessoku.Provide(NewMessagingService)), // Parallel execution
    kessoku.Provide(NewUserService),                     // Depends on database/cache
    kessoku.Provide(NewApp),                             // Final assembly
)
```

### Generated Code with Context Injection

```go
// Context automatically injected for async operations
func InitializeApp(ctx context.Context) (*App, error) {
    var (
        databaseService   *DatabaseService
        cacheService      *CacheService
        messagingService  *MessagingService
        // ... other variables
    )
    
    eg, ctx := errgroup.WithContext(ctx)
    
    // Parallel execution with context cancellation
    eg.Go(func() error {
        var err error
        databaseService, err = kessoku.Async(kessoku.Provide(NewDatabaseService)).Fn()()
        return err
    })
    
    eg.Go(func() error {
        cacheService = kessoku.Async(kessoku.Provide(NewCacheService)).Fn()()
        return nil
    })
    
    // Wait for completion with error handling
    if err := eg.Wait(); err != nil {
        return nil, err
    }
    
    return app, nil
}
```

### Usage with Context

```go
func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Context passed automatically to async providers
    app, err := InitializeApp(ctx)
    if err != nil {
        log.Fatal("Failed to initialize app:", err)
    }
    
    app.Run()
}
```

## CLI Usage

```bash
# Generate DI code for specific files
go tool kessoku kessoku.go

# Multiple files
go tool kessoku file1.go file2.go

# Using go generate (recommended)
go generate ./...

# Show version
go tool kessoku --version
```

### Options

- `-l, --log-level` - Log level (debug, info, warn, error)
- `-v, --version` - Show version information

## API Reference

For detailed API documentation, see the [Go Reference](https://pkg.go.dev/github.com/mazrean/kessoku).

### Quick Reference

- **`kessoku.Inject[T](name, ...providers)`** - Declares an injector function
- **`kessoku.Provide(fn)`** - Wraps a provider function for dependency injection
- **`kessoku.Async(provider)`** - Enables parallel execution of independent providers
- **`kessoku.Bind[I](provider)`** - Binds an interface to its implementation
- **`kessoku.Value(val)`** - Provides a constant value
- **`kessoku.Arg[T](name)`** - Declares a runtime argument

For complete documentation, examples, and detailed function signatures, visit the [Go Reference](https://pkg.go.dev/github.com/mazrean/kessoku).

## Examples

See the [examples/](./examples/) directory for complete working examples:

- **[basic/](./examples/basic/)** - Simple synchronous dependency injection
- **[async_parallel/](./examples/async_parallel/)** - Parallel execution of independent async providers
- **[complex_async/](./examples/complex_async/)** - Complex async dependency chains with coordination
- **[sets/](./examples/sets/)** - Using value sets for configuration
- **[cross_package/](./examples/cross_package/)** - Cross-package dependency injection

## Development

For development guidelines, building, testing, and contributing instructions, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
