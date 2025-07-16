# Kessoku

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)

A dependency injection code generator for Go, similar to [google/wire](https://github.com/google/wire). Kessoku generates Go code for compile-time dependency injection, eliminating runtime reflection overhead.

## Features

**🚀 Parallel Processing** - Execute independent providers concurrently with `kessoku.Async()` while maintaining dependency order and automatic context injection

**⚡ Compile-time Optimization** - Zero runtime overhead with static code generation, full type safety, and optimal performance

**🔧 Enhanced Developer Experience** - Automatic error handling, cycle detection, and seamless Go generate integration

## Installation

**Recommended: Go Tool**

```bash
go get -tool github.com/mazrean/kessoku/cmd/kessoku@latest
```

This installs kessoku as a Go tool, making it available via `go tool kessoku`.

<details>
<summary>Direct Install</summary>

```bash
go install github.com/mazrean/kessoku/cmd/kessoku@latest
```
</details>

<details>
<summary>From Releases</summary>

Download the latest binary from the [releases page](https://github.com/mazrean/kessoku/releases).
</details>

<details>
<summary>Via Homebrew</summary>

```bash
brew install mazrean/tap/kessoku
```
</details>

## Quick Start

Experience Kessoku's **parallel processing power** in 2 simple steps:

### 1. Define Your Services with Async Providers

```go
//go:generate go tool kessoku $GOFILE

package main

import (
    "time"
    "github.com/mazrean/kessoku"
)

// Slow services that can run in parallel
func NewDatabaseService() (*DatabaseService, error) {
    time.Sleep(200 * time.Millisecond) // Simulate slow DB connection
    return &DatabaseService{}, nil
}

func NewCacheService() *CacheService {
    time.Sleep(150 * time.Millisecond) // Simulate slow cache connection
    return &CacheService{}
}

func NewMessagingService() *MessagingService {
    time.Sleep(100 * time.Millisecond) // Simulate slow messaging setup
    return &MessagingService{}
}

// Declare parallel execution with kessoku.Async()
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDatabaseService)),   // 200ms
    kessoku.Async(kessoku.Provide(NewCacheService)),      // 150ms  
    kessoku.Async(kessoku.Provide(NewMessagingService)),  // 100ms
    kessoku.Provide(NewApp),                              // Waits for all
)
```

### 2. Generate and Get Dramatic Performance Boost

```bash
go generate ./...
```

Kessoku generates **optimized parallel code** with automatic context injection and synchronization.

## 🚀 **Performance Comparison**

| Approach | Execution Time | Performance |
|----------|---------------|-------------|
| **Sequential** (google/wire style) | `200ms + 150ms + 100ms = 450ms` | ⏱️ Slow |
| **Parallel** (Kessoku) | `max(200ms, 150ms, 100ms) = 200ms` | ⚡ **2.25x Faster** |

### **Real-world Impact**
- **Application Startup**: 450ms → 200ms (**55% faster**)
- **Test Suite**: Faster dependency setup means faster tests
- **Development**: Quick feedback loop during development

```go
func main() {
    // Automatically gets context parameter for async operations
    app, err := InitializeApp(context.Background())
    if err != nil {
        log.Fatal("Failed to initialize:", err)
    }
    
    app.Run() // Ready in 200ms instead of 450ms!
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
