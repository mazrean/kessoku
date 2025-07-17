# Kessoku

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)

**Kessoku is a dependency injection code generator for Go that enables parallel execution of independent providers.** Similar to google/wire, but with automatic parallelization that can improve application startup performance by up to 2.25x.

## How It Works

Kessoku analyzes your dependency graph and identifies providers that can run independently. When you mark providers with `kessoku.Async()`, the generator creates code that executes them in parallel using goroutines and `errgroup`, while maintaining proper dependency ordering.

**Key principle:** Providers with no dependencies between them can execute concurrently, reducing total initialization time to the duration of the longest individual operation.

## Performance Impact

**Sequential execution (traditional):**
```
DB Connection:     200ms  ⏳
Cache Setup:       150ms  ⏳  
API Initialization: 100ms  ⏳
────────────────────────────
Total Wait Time:   450ms
```

**Parallel execution (Kessoku):**
```
DB + Cache + API:  200ms  (all concurrent)
────────────────────────────
Same Result:       200ms  (55% improvement)
```

**Important:** Providers with dependencies still execute in correct order, but independent providers run concurrently.

### When to Use Kessoku

**Ideal for applications with:**
- Multiple slow initialization operations (database connections, API clients, file I/O)
- Independent service setup that currently runs sequentially
- Existing google/wire usage that could benefit from parallelization
- Performance-critical startup requirements (microservices, Lambda functions)

**Consider alternatives if:**
- Your app has very few dependencies (< 3 providers)
- Most providers are already fast (< 50ms each)
- Startup time is not a performance concern

### Key Benefits

- **Measurable Performance Gains** - Up to 2.25x faster startup demonstrated in examples
- **Familiar API** - Compatible with google/wire patterns and conventions
- **Easy Migration** - Minimal changes required from existing google/wire projects
- **Automatic Coordination** - Handles dependency ordering and error propagation in parallel execution

### Trade-offs

- **Memory overhead** - Uses more goroutines during initialization
- **Complexity** - Generated code is more complex than sequential versions
- **Context requirement** - Async providers require context.Context parameter

## Performance Comparison

### Sequential Execution (Traditional)
```
┌─────────────────────────────────────────────────┐
│  DB Setup   │ Cache Init │  API Load  │ Ready!   │
│    200ms    │   150ms    │   100ms    │          │
└─────────────────────────────────────────────────┘
                    Total: 450ms
```

### Parallel Execution (Kessoku)
```
┌─────────────────────────┐
│  DB Setup   │           │
│ Cache Init  │  Ready!   │
│  API Load   │           │
└─────────────────────────┘
       200ms (2.25x improvement)
```

### Real-World Impact

| Scenario | Before | After | Time Saved |
|----------|--------|-------|------------|
| **Application Startup** | 450ms | 200ms | 250ms per restart |
| **Test Suite Execution** | 20 seconds | 9 seconds | 11 seconds per test run |
| **Development Cycle** | Slow feedback | Fast feedback | Improved iteration speed |
| **Production Deployment** | Higher latency | Lower latency | Better user experience |

**Performance calculation:** The 2.25x improvement (450ms → 200ms) comes from the specific example where three independent 200ms, 150ms, and 100ms operations run in parallel instead of sequentially. Actual improvements depend on your specific provider durations and dependency structure.

**Measured in [examples/async_parallel](./examples/async_parallel/):** Real timing tests show consistent ~2.2x improvement on typical hardware.

## Installation

```bash
go get github.com/mazrean/kessoku
```

Kessoku is now ready to use with `go generate`.

<details>
<summary>Alternative: Standalone Tool</summary>

```bash
go install github.com/mazrean/kessoku/cmd/kessoku@latest
# Then use: kessoku *.go
```
</details>

## Quick Start Example

**Evaluate Kessoku** with this working example:

### Step 1: Install
```bash
go get github.com/mazrean/kessoku
```

### Step 2: Copy This Simple Example
```go
// main.go
//go:generate go tool kessoku $GOFILE
package main

import (
    "fmt"
    "time"
    "context"
    "github.com/mazrean/kessoku"
)

// These services have no dependencies between them, so they can run in parallel
func NewDB() string { time.Sleep(200*time.Millisecond); return "DB Ready" }
func NewCache() string { time.Sleep(150*time.Millisecond); return "Cache Ready" }
func NewAPI() string { time.Sleep(100*time.Millisecond); return "API Ready" }

// kessoku.Async() enables parallel execution for independent providers
var _ = kessoku.Inject[string](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),     // 200ms 
    kessoku.Async(kessoku.Provide(NewCache)),  // 150ms  } All execute concurrently
    kessoku.Async(kessoku.Provide(NewAPI)),    // 100ms 
    // This final provider waits for all async providers to complete
    kessoku.Provide(func(db, cache, api string) string {
        return fmt.Sprintf("App: %s, %s, %s", db, cache, api)
    }),
)

func main() {
    start := time.Now()
    app, _ := InitializeApp(context.Background())
    fmt.Printf("%s in %v (normally 450ms)\n", app, time.Since(start))
}
```

### Step 3: Run the Example
```bash
go generate && go run main.go
# Output: App ready in ~200ms (normally 450ms)
# Result: 55% faster startup
```

This demonstrates parallel dependency injection reducing startup time from 450ms to 200ms.

## When Parallel Execution Applies

**✅ Can run in parallel:**
- Providers with no dependencies between them
- Independent service initializations (DB, Cache, API clients)
- Configuration loading from different sources

**❌ Cannot run in parallel:**
- Providers that depend on each other's output
- Services that must be initialized in specific order

Example with dependencies:
```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),                    // Runs first
    kessoku.Async(kessoku.Provide(NewDatabase)),   // Parallel (needs Config)
    kessoku.Async(kessoku.Provide(NewCache)),      // Parallel (needs Config)  
    kessoku.Provide(NewApp),                       // Waits for DB and Cache
)
```

Kessoku automatically handles the dependency ordering while maximizing parallelization opportunities.

## CLI Usage

```
Usage: kessoku <files> ... [flags]

A dependency injection code generator for Go, similar to google/wire

Arguments:
  <files> ...    Go files to process

Flags:
  -h, --help                Show context-sensitive help.
  -l, --log-level="info"    Log level
  -v, --version             Show version and exit.
```

**Common usage**
```bash
go tool kessoku kessoku.go        # Process single file
go tool kessoku *.go              # Process multiple files
go generate ./...                 # Using go generate (recommended)
```

## API Reference

For detailed API documentation, see the [Go Reference](https://pkg.go.dev/github.com/mazrean/kessoku).

### Quick Reference

- **`kessoku.Inject[T](name, ...providers)`** - Declares an injector function
- **`kessoku.Provide(fn)`** - Wraps a provider function for dependency injection
- **`kessoku.Async(provider)`** - Enables parallel execution of independent providers
- **`kessoku.Bind[I](provider)`** - Binds an interface to its implementation
- **`kessoku.Value(val)`** - Provides a constant value
- **`kessoku.Set(...providers)`** - Groups related providers together as a reusable set

## Migration from google/wire

**For existing google/wire users:** Migration requires minimal changes:

### Before (google/wire)
```go
//+build wireinject

//go:generate wire
func InitializeApp() (*App, error) {
    wire.Build(NewDB, NewCache, NewApp)
    return &App{}, nil
}
```

### After (Kessoku)
```go
//go:generate go tool kessoku $GOFILE

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),    // Now executes in parallel
    kessoku.Async(kessoku.Provide(NewCache)), // Now executes in parallel
    kessoku.Provide(NewApp),
)
```

**Required changes:**
1. Replace `wire.Build()` with `kessoku.Inject[T]()`
2. Wrap slow providers with `kessoku.Async()` for parallel execution
3. Update `//go:generate` directive
4. Result: Up to 2.25x faster startup performance

## Comparison with Alternatives

| Feature | Kessoku | google/wire | uber/fx | sarulabs/di |
|---------|---------|-------------|---------|-------------|
| **Execution Model** | Parallel + Sequential | Sequential only | Runtime | Runtime |
| **Performance** | High (parallel) | Medium | Low (reflection) | Low (reflection) |
| **Learning Curve** | Low (wire-like) | Low | Medium | Medium |
| **Code Generation** | Yes | Yes | No | No |
| **Type Safety** | Compile-time | Compile-time | Runtime | Runtime |
| **Best For** | Performance-critical apps | Simple DI | Complex lifecycles | Runtime flexibility |

**Choose Kessoku when:** Startup performance matters and you have independent, slow providers.
**Choose google/wire when:** You need simple DI without performance requirements.
**Choose uber/fx when:** You need complex dependency lifecycles and shutdown hooks.
**Choose runtime DI when:** You need dynamic dependency resolution.

## Examples

See the [examples/](./examples/) directory for complete working examples:

- **[basic/](./examples/basic/)** - Simple dependency injection with in-memory database and user operations
- **[async_parallel/](./examples/async_parallel/)** - Parallel execution demonstrating 2.2x performance improvement
- **[sets/](./examples/sets/)** - Provider organization with Set patterns (basic, reusable, nested)

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
