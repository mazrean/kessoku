---
name: kessoku-di
description: Kessoku compile-time DI with parallel initialization for Go. Use when writing or debugging kessoku providers/injectors, enabling async dependencies, migrating from google/wire, or fixing go:generate/codegen issues in Go services.
---

# Kessoku: Parallel DI for Go

Kessoku is a compile-time dependency injection library that speeds up Go application startup through **parallel initialization**. Unlike google/wire (sequential), kessoku runs independent providers concurrently.

**Use this skill when** you are building or debugging kessoku injectors/providers, wiring async dependencies, migrating from google/wire, or troubleshooting `go tool kessoku` generation problems.

**Supporting files:** [PATTERNS.md](references/PATTERNS.md) for examples, [MIGRATION.md](references/MIGRATION.md) for wire migration, [TROUBLESHOOTING.md](references/TROUBLESHOOTING.md) for common errors.

## When to Use Kessoku

- **Slow startup**: Multiple DB connections, API clients, or cache initialization
- **Cold start optimization**: Serverless/Lambda functions timing out
- **google/wire alternative**: Want compile-time DI with parallel execution

## Quick Start

```go
//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),     // Parallel
    kessoku.Async(kessoku.Provide(NewCache)),  // Parallel
    kessoku.Provide(NewApp),                   // Waits for deps
)
```

Run: `go generate ./...` → generates `*_band.go`

## API Reference

| API | Syntax | Purpose |
|-----|--------|---------|
| **Inject** | `var _ = kessoku.Inject[T]("Name", ...)` | Define injector function |
| **Provide** | `kessoku.Provide(NewFn)` | Wrap provider function |
| **Async** | `kessoku.Async(kessoku.Provide(...))` | Enable parallel execution |
| **Bind** | `kessoku.Bind[Interface](provider)` | Interface→implementation |
| **Value** | `kessoku.Value(v)` | Inject constant value |
| **Set** | `kessoku.Set(providers...)` | Group providers |
| **Struct** | `kessoku.Struct[T]()` | Expand struct fields as deps |

## Common Patterns

### Basic Injector

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDB),
    kessoku.Provide(NewApp),
)
// Generates: func InitializeApp() *App
```

### Parallel Initialization (Async)

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),      // }
    kessoku.Async(kessoku.Provide(NewCache)),   // } Run in parallel
    kessoku.Async(kessoku.Provide(NewClient)),  // }
    kessoku.Provide(NewApp),
)
// Generates: func InitializeApp(ctx context.Context) (*App, error)
```

### Interface Binding

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
    kessoku.Provide(NewApp),  // Receives Repository interface
)
```

### Provider Sets (Reusability)

```go
var DatabaseSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Async(kessoku.Provide(NewDB)),
)

var _ = kessoku.Inject[*App]("InitializeApp",
    DatabaseSet,
    kessoku.Provide(NewApp),
)
```

### Struct Field Expansion

```go
var _ = kessoku.Inject[*DB](
    "InitializeDB",
    kessoku.Provide(NewConfig),   // Returns *Config{Host, Port}
    kessoku.Struct[*Config](),    // Expands to Host, Port fields
    kessoku.Provide(NewDB),       // Receives Host, Port separately
)
```

## Key Points

- **No build tag needed**: Unlike wire's `//go:build wireinject`
- **Async requires context**: Generated function gets `context.Context` parameter
- **Error propagation**: Providers returning `error` make injector return `error`
- **Cleanup functions**: Providers can return `func()` for cleanup

## Support Files

For detailed information, see:
- [PATTERNS.md](references/PATTERNS.md) - Detailed patterns, examples, best practices
- [MIGRATION.md](references/MIGRATION.md) - Wire migration guide
- [TROUBLESHOOTING.md](references/TROUBLESHOOTING.md) - Common errors and solutions

## Resources

- Repository: https://github.com/mazrean/kessoku
- Documentation: https://pkg.go.dev/github.com/mazrean/kessoku
