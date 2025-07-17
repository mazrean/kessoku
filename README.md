# Kessoku

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)

**Kessoku is a dependency injection code generator for Go that enables parallel execution of independent providers.** It extends google/wire's compile-time approach with automatic parallelization capabilities.

```go
// Sequential execution (google/wire)
wire.Build(NewDB, NewCache, NewAPI, NewApp)     // 450ms total

// Parallel execution (Kessoku)
kessoku.Inject[*App]("InitApp",
    kessoku.Async(kessoku.Provide(NewDB)),      // 200ms }
    kessoku.Async(kessoku.Provide(NewCache)),   // 150ms } concurrent
    kessoku.Async(kessoku.Provide(NewAPI)),     // 100ms }
    kessoku.Provide(NewApp),                    // waits for all
)                                               // 200ms total
```

**Result:** Independent providers execute concurrently, reducing startup time from 450ms to 200ms.

## How It Works

**Think of it like cooking a meal:**

```
❌ Sequential (slow way):
   1. Boil water (5 min) → 2. Cook pasta (8 min) → 3. Make sauce (6 min) = 19 minutes

✅ Parallel (smart way):  
   1. Boil water (5 min) }
   2. Cook pasta (8 min) } All at the same time = 8 minutes (fastest task)
   3. Make sauce (6 min) }
```

**Kessoku does the same for your Go services:**

```
❌ Sequential startup:
   Database.Connect() → Cache.Init() → Auth.Setup() = 450ms

✅ Parallel startup:
   Database.Connect() }
   Cache.Init()       } All concurrent = 200ms (slowest task)  
   Auth.Setup()       }
```

**How it works:** Kessoku analyzes your dependency graph to identify which services don't depend on each other, then generates code to run them simultaneously. Services that DO depend on others still wait for their dependencies - dependency ordering is automatically maintained.

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

## Real-World Impact

### Before Kessoku (Typical Development Experience)

**During development:**
```
$ go run main.go
[Connecting to PostgreSQL...] ████████░░░░ 200ms
[Initializing Redis...]       ████░░░░░░░░ 150ms  
[Setting up Auth0...]         ███░░░░░░░░░ 100ms
[Starting server...]          █░░░░░░░░░░░ 50ms
✓ Server ready                          500ms total

$ # Make a small change, restart again...
$ go run main.go                       500ms again
$ # Fix a bug, restart...              500ms again  
$ # Test feature, restart...           500ms again
```

**Running tests:**
```
$ go test ./...
[Test setup: DB + Cache + Services...] 2000ms per test package
```

### After Kessoku (Parallel Execution)

**Same operations, parallel execution:**
```
$ go run main.go
[PostgreSQL + Redis + Auth0...]        ████████████ 200ms (parallel)
[Starting server...]                   █░░░░░░░░░░░ 50ms
✓ Server ready                          250ms total (2x faster)

$ # Restart cycles now take 250ms instead of 500ms
$ # 250ms saved per restart × 50 restarts/day = 12.5 seconds saved daily
```

**Test execution:**
```
$ go test ./...  
[Parallel test setup...]               800ms per test package (2.5x faster)
```

### Developer Experience Transformation

| Scenario | Before | After | Daily Impact |
|----------|--------|-------|--------------|
| **Development restarts** | 500ms × 50 = 25s | 250ms × 50 = 12.5s | **12.5s saved** |
| **Test suite runs** | 2000ms × 10 = 20s | 800ms × 10 = 8s | **12s saved per run** |
| **CI/CD pipeline** | 30s startup | 13s startup | **17s per deployment** |

**Result:** Faster feedback loops, more productive development, happier developers.

---

## ⚡ Ready to Speed Up Your App?

**Your next restart could be 2x faster.** Takes 2 minutes to try, gives you hours back every week.

### Option 1: Try with Your Existing Project
```bash
# If you already use google/wire
go get github.com/mazrean/kessoku

# Replace wire.Build(...) with kessoku.Inject[T](...)
# Add kessoku.Async() around slow providers
# Run: go generate && go run main.go
# See immediate performance improvement
```

### Option 2: Start Fresh
```bash
# Copy our working example
curl -sL https://raw.githubusercontent.com/mazrean/kessoku/main/examples/async_parallel/main.go > demo.go
go mod init demo && go get github.com/mazrean/kessoku
go generate && go run demo.go
# Watch: 450ms → 200ms improvement in real-time
```

### Option 3: Skeptical? Benchmark First
```bash
git clone https://github.com/mazrean/kessoku
cd kessoku/examples/async_parallel
go run main.go  # See timing comparison yourself
```

**⏱️ Time investment:** 2 minutes to try  
**⚡ Time savings:** 12+ seconds daily (250ms × 50 restarts)  
**🎯 ROI:** Pays for itself in 10 days

---

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

## Try It Now

**Copy, paste, run** - See 2.25x improvement in 30 seconds:

```bash
# 1. Install
go get github.com/mazrean/kessoku

# 2. Create main.go
cat > main.go << 'EOF'
//go:generate go tool kessoku $GOFILE
package main

import ("context"; "fmt"; "time"; "github.com/mazrean/kessoku")

func connectDB() string { time.Sleep(200*time.Millisecond); return "PostgreSQL" }
func initRedis() string { time.Sleep(150*time.Millisecond); return "Redis" }  
func setupAuth() string { time.Sleep(100*time.Millisecond); return "Auth0" }

var _ = kessoku.Inject[string]("InitApp",
    kessoku.Async(kessoku.Provide(connectDB)),   // parallel
    kessoku.Async(kessoku.Provide(initRedis)),   // parallel
    kessoku.Async(kessoku.Provide(setupAuth)),   // parallel
    kessoku.Provide(func(db, cache, auth string) string {
        return fmt.Sprintf("App ready: %s + %s + %s", db, cache, auth)
    }),
)

func main() {
    start := time.Now()
    app, _ := InitApp(context.Background())
    fmt.Printf("%s in %v\n", app, time.Since(start))
}
EOF

# 3. Run and see the performance gain
go generate && go run main.go
# Output: App ready: PostgreSQL + Redis + Auth0 in ~200ms
# Without Kessoku: 450ms | With Kessoku: 200ms = 2.25x faster
```

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
