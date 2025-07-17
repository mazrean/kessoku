# Kessoku ⚡

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)

## ⚡ Make Your Go App **2.25x Faster**

**Current situation:** Your services start one by one
```
DB Connection:     200ms  ⏳
Cache Setup:       150ms  ⏳  
API Initialization: 100ms  ⏳
────────────────────────────
Total Wait Time:   450ms  😴
```

**With Kessoku:** All services start together

## ⚡ Kessoku Makes It **2.25x FASTER**

```
DB + Cache + API:  200ms  🚀 (parallel execution)
────────────────────────────
Same Result:       200ms  ⚡ 55% faster!
```

**Parallel dependency injection** - Execute independent services simultaneously instead of waiting sequentially.

### Why Developers Love Kessoku

- 🚀 **Instant Gratification** - See 2x speed boost immediately
- 🛠️ **Zero Learning Curve** - If you know google/wire, you know Kessoku
- 🔧 **Drop-in Replacement** - Migrate from google/wire in minutes
- ⚡ **Smart Parallelization** - Automatic dependency coordination

## 📊 Real Performance Impact

### Before Kessoku (Sequential) 😴
```
┌─────────────────────────────────────────────────┐
│  DB Setup   │ Cache Init │  API Load  │ Ready!   │
│    200ms    │   150ms    │   100ms    │          │
└─────────────────────────────────────────────────┘
                    Total: 450ms 🐌
```

### After Kessoku (Parallel) ⚡
```
┌─────────────────────────┐
│  DB Setup   │           │
│ Cache Init  │  Ready!   │
│  API Load   │           │
└─────────────────────────┘
       200ms ⚡ **2.25x faster!**
```

### What This Means For You

| Scenario | Before | After | You Save |
|----------|--------|-------|----------|
| 🚀 **App Startup** | 450ms | 200ms | **250ms every restart** |
| 🧪 **Test Runs** | 20 seconds | 9 seconds | **11 seconds per test** |
| 🔄 **Development** | Slow feedback | Instant | **More iterations per hour** |
| 📱 **User Experience** | Sluggish | Snappy | **Happy users** |

## 📦 Installation

```bash
go get github.com/mazrean/kessoku
```

**That's it!** Kessoku is now ready to use with `go generate`.

<details>
<summary>Alternative: Standalone Tool</summary>

```bash
go install github.com/mazrean/kessoku/cmd/kessoku@latest
# Then use: kessoku *.go
```
</details>

## 🚀 30-Second Speed Boost

**Try Kessoku NOW** - Copy, paste, and feel the speed:

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

// Simulate slow services
func NewDB() string { time.Sleep(200*time.Millisecond); return "DB Ready" }
func NewCache() string { time.Sleep(150*time.Millisecond); return "Cache Ready" }
func NewAPI() string { time.Sleep(100*time.Millisecond); return "API Ready" }

// 🚀 Magic happens here - all run in parallel!
var _ = kessoku.Inject[string](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),     // 200ms 
    kessoku.Async(kessoku.Provide(NewCache)),  // 150ms  } All parallel!
    kessoku.Async(kessoku.Provide(NewAPI)),    // 100ms 
    kessoku.Provide(func(db, cache, api string) string {
        return fmt.Sprintf("App: %s, %s, %s", db, cache, api)
    }),
)

func main() {
    start := time.Now()
    app, _ := InitializeApp(context.Background())
    fmt.Printf("⚡ %s in %v (normally 450ms)\n", app, time.Since(start))
}
```

### Step 3: See the Magic
```bash
go generate && go run main.go
# ⚡ App ready in ~200ms (normally 450ms)
# 🎉 55% faster startup!
```

**That's it!** You just made your app **2.25x faster** with parallel dependency injection.

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

## 🔄 Migrate from google/wire

**Already using google/wire?** Migrate in **2 minutes**:

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
    kessoku.Async(kessoku.Provide(NewDB)),    // 🚀 Now parallel!
    kessoku.Async(kessoku.Provide(NewCache)), // 🚀 Now parallel!
    kessoku.Provide(NewApp),
)
```

**Changes needed:**
1. Replace `wire.Build()` with `kessoku.Inject[T]()`
2. Wrap slow providers with `kessoku.Async()`
3. Update `//go:generate` directive
4. **Boom! 2x faster startup**

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
