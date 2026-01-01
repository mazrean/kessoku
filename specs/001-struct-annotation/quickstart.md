# Quickstart: Struct Annotation for Field Expansion

**Feature Branch**: `001-struct-annotation`
**Date**: 2026-01-01

## What is `kessoku.Struct[T]()`?

The `Struct` annotation automatically expands all exported fields of a struct as individual dependencies. Instead of creating separate provider functions for each field, declare the struct once and let kessoku make its fields available for injection.

---

## Before & After

### Before: Manual Field Providers

```go
type Config struct {
    DBHost string
    DBPort int
    Debug  bool
}

func NewConfig() *Config { return &Config{...} }

// Manual providers for each field
func DBHost(c *Config) string { return c.DBHost }
func DBPort(c *Config) int    { return c.DBPort }
func Debug(c *Config) bool    { return c.Debug }

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Provide(DBHost),
    kessoku.Provide(DBPort),
    kessoku.Provide(Debug),
    kessoku.Provide(NewApp),
)
```

### After: Using `kessoku.Struct[T]()`

```go
type Config struct {
    DBHost string
    DBPort int
    Debug  bool
}

func NewConfig() *Config { return &Config{...} }

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Struct[*Config](),  // Expands all fields automatically!
    kessoku.Provide(NewApp),
)
```

---

## Basic Usage

### Step 1: Define Your Struct

```go
package main

type DatabaseConfig struct {
    Host     string
    Port     int
    Username string
    Password string
}
```

### Step 2: Create a Provider for the Struct

```go
func NewDatabaseConfig() *DatabaseConfig {
    return &DatabaseConfig{
        Host:     os.Getenv("DB_HOST"),
        Port:     5432,
        Username: os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASS"),
    }
}
```

### Step 3: Use `kessoku.Struct[T]()` in Inject

```go
//go:generate go tool kessoku $GOFILE

var _ = kessoku.Inject[*Database](
    "InitializeDatabase",
    kessoku.Provide(NewDatabaseConfig),
    kessoku.Struct[*DatabaseConfig](),  // Host, Port, Username, Password now available
    kessoku.Provide(NewDatabase),       // Can receive string, int as dependencies
)
```

### Step 4: Run Code Generation

```bash
go generate ./...
```

---

## Integration Examples

### With Async Providers

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewConfig)),  // Config created asynchronously
    kessoku.Struct[*Config](),                   // Fields extracted after async completes
    kessoku.Provide(NewApp),
)
```

### With Sets

```go
var ConfigSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Struct[*Config](),
)

var _ = kessoku.Inject[*App](
    "InitializeApp",
    ConfigSet,
    kessoku.Provide(NewApp),
)
```

### With Bind

```go
type Logger interface { Log(msg string) }
type FileLogger struct{}

type Config struct {
    Logger *FileLogger
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Struct[*Config](),                    // Provides *FileLogger
    kessoku.Bind[Logger](kessoku.Provide(func(l *FileLogger) Logger { return l })),
    kessoku.Provide(NewApp),
)
```

---

## Embedded Fields

Embedded (anonymous) fields are provided as their type:

```go
type BaseConfig struct {
    Debug bool
}

type AppConfig struct {
    BaseConfig  // Embedded - provides BaseConfig
    Name string
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewAppConfig),
    kessoku.Struct[*AppConfig](),  // Provides: BaseConfig, string
    kessoku.Provide(NewApp),
)
```

**Note**: Nested fields of embedded structs are NOT recursively expanded. To access `AppConfig.BaseConfig.Debug`, use a separate `kessoku.Struct[BaseConfig]()`.

---

## Important Notes

### Type Matching

- `kessoku.Struct[*Config]()` requires a `*Config` provider
- `kessoku.Struct[Config]()` requires a `Config` provider
- Type mismatch results in an error

### Unexported Fields

Unexported fields (lowercase names) are ignored:

```go
type Config struct {
    Host    string  // Exported - available for injection
    port    int     // Unexported - ignored
    debug   bool    // Unexported - ignored
}
```

### Type Conflicts

If multiple fields have the same type, kessoku's existing type conflict detection applies:

```go
type Config struct {
    Host     string  // string type
    Username string  // string type - conflict!
}

// Error: multiple providers provide string
```

### Field Ordering

Fields are generated in alphabetical order for deterministic output:

```go
type Config struct {
    Zebra int
    Apple string
    Mango bool
}

// Generated order: Apple, Mango, Zebra
```

---

## Error Messages

| Situation | Error Message |
|-----------|---------------|
| Non-struct type | "not a struct type: `<type>`" |
| Missing struct provider | "no provider for type `<type>`" |
| Type mismatch (pointer/value) | "type mismatch: expected `<expected>`, got `<actual>`" |
| Duplicate types | "multiple providers provide `<type>`" |

**Note on type matching**: `kessoku.Struct[*Config]()` requires exactly a `*Config` provider. If you have a `Config` (non-pointer) provider instead, you'll get a type mismatch error. This is intentional - kessoku requires exact type matching for safety.

---

## Complete Example

```go
package main

import "github.com/mazrean/kessoku"

//go:generate go tool kessoku $GOFILE

type Config struct {
    DBHost string
    DBPort int
    Debug  bool
}

type Database struct {
    host string
    port int
}

func NewConfig() *Config {
    return &Config{
        DBHost: "localhost",
        DBPort: 5432,
        Debug:  true,
    }
}

func NewDatabase(host string, port int) *Database {
    return &Database{host: host, port: port}
}

var _ = kessoku.Inject[*Database](
    "InitializeDatabase",
    kessoku.Provide(NewConfig),
    kessoku.Struct[*Config](),
    kessoku.Provide(NewDatabase),
)

func main() {
    db := InitializeDatabase()
    // db is initialized with config.DBHost and config.DBPort
}
```
