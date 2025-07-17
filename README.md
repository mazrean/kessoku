# Kessoku

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)

**Kessoku speeds up Go application startup by 2.25x through parallel dependency injection.**

```go
// Before: Sequential (google/wire)
wire.Build(NewDB, NewCache, NewAPI, NewApp)     // 450ms

// After: Parallel (Kessoku)  
kessoku.Inject[*App]("InitApp",
    kessoku.Async(kessoku.Provide(NewDB)),      // 200ms }
    kessoku.Async(kessoku.Provide(NewCache)),   // 150ms } concurrent
    kessoku.Async(kessoku.Provide(NewAPI)),     // 100ms }
    kessoku.Provide(NewApp),                    // waits for all
)                                               // 200ms total = 2.25x faster
```

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

## Why Kessoku?

**Your development cycle:**
```
❌ Sequential startup: DB(200ms) → Cache(150ms) → Auth(100ms) = 450ms
✅ Parallel startup:   DB + Cache + Auth together = 200ms (2.25x faster)

Result: 250ms saved per restart × 50 restarts/day = 12.5 seconds saved daily
```

**Perfect for:**
- Multiple slow services (databases, APIs, external connections)
- Performance-critical environments (microservices, serverless)
- Existing google/wire projects wanting speed improvements

**Trade-offs:** Slightly more memory during startup, requires context.Context parameter

## Quick Start

**Copy, paste, see 2.25x improvement in 30 seconds:**

```bash
# Install and create demo
go get github.com/mazrean/kessoku

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

# Run and see the improvement
go generate && go run main.go
# Output: App ready: PostgreSQL + Redis + Auth0 in ~200ms
# Without Kessoku: would be 450ms
```

## Migration from google/wire

**Already using google/wire?** Upgrade in 2 minutes:

```go
// Before
//go:generate wire
func InitApp() (*App, error) {
    wire.Build(NewDB, NewCache, NewApp)
    return &App{}, nil
}

// After  
//go:generate go tool kessoku $GOFILE
var _ = kessoku.Inject[*App]("InitApp",
    kessoku.Async(kessoku.Provide(NewDB)),    // now parallel
    kessoku.Async(kessoku.Provide(NewCache)), // now parallel
    kessoku.Provide(NewApp),
)
```

**Changes:** Replace `wire.Build()` → `kessoku.Inject[T]()`, add `kessoku.Async()` for slow providers

## API Reference

**Core functions:**
- **`kessoku.Inject[T](name, ...providers)`** - Declares an injector function
- **`kessoku.Async(provider)`** - Enables parallel execution  
- **`kessoku.Provide(fn)`** - Wraps a provider function
- **`kessoku.Set(...providers)`** - Groups providers for reuse
- **`kessoku.Bind[Interface](provider)`** - Interface binding
- **`kessoku.Value(val)`** - Constant value injection

**Dependency rules:** Independent providers run in parallel, dependent providers wait automatically.

---

## Advanced Usage

**Installation:** `go get github.com/mazrean/kessoku`

**vs Alternatives:**
- **Kessoku:** Parallel execution, 2.25x faster startup
- **google/wire:** Sequential only, simpler but slower  
- **uber/fx:** Runtime DI, complex lifecycles, reflection overhead

**Examples:** See [examples/](./examples/) - basic, async_parallel, sets

**Full docs:** [pkg.go.dev/github.com/mazrean/kessoku](https://pkg.go.dev/github.com/mazrean/kessoku)
