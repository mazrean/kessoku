# Kessoku Design Patterns

Detailed patterns and examples for kessoku dependency injection.

## Provider Patterns

### Basic Provider

A provider is any function that returns a value:

```go
func NewConfig() *Config {
    return &Config{Host: "localhost", Port: 8080}
}

func NewDB(cfg *Config) *DB {
    return &DB{host: cfg.Host, port: cfg.Port}
}

var _ = kessoku.Inject[*DB](
    "InitializeDB",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDB),
)
// Generates: func InitializeDB() *DB
```

### Provider with Error

When a provider returns an error, the generated injector also returns error:

```go
func NewDB(cfg *Config) (*DB, error) {
    db, err := sql.Open("postgres", cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("failed to open db: %w", err)
    }
    return &DB{db: db}, nil
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDB),
    kessoku.Provide(NewApp),
)
// Generates: func InitializeApp() (*App, error)
```

### Provider with Cleanup

Return a cleanup function for resource management:

```go
func NewDB() (*DB, func(), error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, nil, err
    }
    cleanup := func() { db.Close() }
    return &DB{db: db}, cleanup, nil
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewDB),
    kessoku.Provide(NewApp),
)
// Generates: func InitializeApp() (*App, func(), error)
```

Usage:

```go
app, cleanup, err := InitializeApp()
if err != nil {
    log.Fatal(err)
}
defer cleanup()
```

### Multiple Return Values

Providers can return multiple values (all become available as dependencies):

```go
func NewDBAndCache(cfg *Config) (*DB, *Cache) {
    return &DB{...}, &Cache{...}
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDBAndCache),  // Provides both *DB and *Cache
    kessoku.Provide(NewApp),         // Can receive *DB and *Cache
)
```

## Async Patterns

### Basic Async

Mark slow providers with Async for parallel execution:

```go
func NewDB(ctx context.Context) (*DB, error) {
    // Slow operation - database connection
    return connectDB(ctx)
}

func NewCache(ctx context.Context) (*Cache, error) {
    // Slow operation - cache connection
    return connectCache(ctx)
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),    // }
    kessoku.Async(kessoku.Provide(NewCache)), // } Run concurrently
    kessoku.Provide(NewApp),                  // Waits for both
)
// Generates: func InitializeApp(ctx context.Context) (*App, error)
```

### Async with Context

Async providers can optionally receive `context.Context`:

```go
// With context - for cancellation/timeout
func NewDB(ctx context.Context) (*DB, error) {
    return connectWithContext(ctx)
}

// Without context - still runs in parallel
func NewCache() (*Cache, error) {
    return connectCache()
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewDB)),    // Gets ctx
    kessoku.Async(kessoku.Provide(NewCache)), // No ctx needed
    kessoku.Provide(NewApp),
)
```

### Async Dependency Chains

Dependencies are respected - async only parallelizes independent providers:

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Async(kessoku.Provide(NewConfig)),    // Pool 1
    kessoku.Async(kessoku.Provide(NewDB)),        // Pool 2 (needs Config)
    kessoku.Async(kessoku.Provide(NewCache)),     // Pool 2 (needs Config)
    kessoku.Provide(NewApp),                      // Pool 3 (needs DB, Cache)
)
// Config runs first, then DB+Cache in parallel, then App
```

## Interface Binding Patterns

### Basic Binding

Bind interface to implementation for testability:

```go
type Repository interface {
    FindUser(id int) (*User, error)
}

type PostgresRepo struct{ db *sql.DB }

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
    return &PostgresRepo{db: db}
}

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewDB),
    kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
    kessoku.Provide(NewApp),  // Receives Repository, not *PostgresRepo
)
```

### Multiple Interface Bindings

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Bind[UserRepository](kessoku.Provide(NewPostgresUserRepo)),
    kessoku.Bind[OrderRepository](kessoku.Provide(NewPostgresOrderRepo)),
    kessoku.Provide(NewApp),
)
```

### Binding Value to Interface

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Bind[Logger](kessoku.Value(log.Default())),
    kessoku.Provide(NewApp),
)
```

## Set Patterns

### Basic Set

Group related providers:

```go
var DatabaseSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDB),
    kessoku.Provide(NewMigrator),
)

var _ = kessoku.Inject[*App](
    "InitializeApp",
    DatabaseSet,
    kessoku.Provide(NewApp),
)
```

### Nested Sets

Sets can include other sets:

```go
var ConfigSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewSecrets),
)

var DatabaseSet = kessoku.Set(
    ConfigSet,  // Include ConfigSet
    kessoku.Async(kessoku.Provide(NewDB)),
)

var _ = kessoku.Inject[*App](
    "InitializeApp",
    DatabaseSet,  // Includes Config, Secrets, DB
    kessoku.Provide(NewApp),
)
```

### Sets with Async

```go
var InfraSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Async(kessoku.Provide(NewDB)),
    kessoku.Async(kessoku.Provide(NewCache)),
    kessoku.Async(kessoku.Provide(NewMQ)),
)

var _ = kessoku.Inject[*App](
    "InitializeApp",
    InfraSet,
    kessoku.Provide(NewApp),
)
```

## Struct Expansion Pattern

### Basic Struct Expansion

Extract struct fields as individual dependencies:

```go
type Config struct {
    DBHost   string
    DBPort   int
    CacheURL string
}

func NewConfig() *Config {
    return &Config{DBHost: "localhost", DBPort: 5432, CacheURL: "redis://..."}
}

func NewDB(host string, port int) *DB {
    return &DB{host: host, port: port}
}

var _ = kessoku.Inject[*DB](
    "InitializeDB",
    kessoku.Provide(NewConfig),   // Provides *Config
    kessoku.Struct[*Config](),    // Expands to DBHost(string), DBPort(int), CacheURL(string)
    kessoku.Provide(NewDB),       // Receives string and int (matched by type)
)
```

**Important**: `kessoku.Struct[T]()` takes NO arguments. It expands ALL exported fields.

### Struct vs Provide

| Pattern | Use When |
|---------|----------|
| `kessoku.Struct[T]()` | Need individual fields as dependencies |
| `kessoku.Provide(NewT)` | Need the struct itself as dependency |

## Value Injection Pattern

### Basic Value

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Value("localhost:5432"),      // string
    kessoku.Value(30 * time.Second),      // time.Duration
    kessoku.Value(true),                  // bool
    kessoku.Provide(NewApp),
)
```

### Value with Interface Binding

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Bind[io.Writer](kessoku.Value(os.Stdout)),
    kessoku.Provide(NewApp),
)
```

## Best Practices

### 1. Use Async for I/O Operations

```go
// Good: Parallelize I/O-bound operations
kessoku.Async(kessoku.Provide(NewDB))       // Network I/O
kessoku.Async(kessoku.Provide(NewRedis))    // Network I/O
kessoku.Async(kessoku.Provide(NewS3Client)) // Network I/O

// Unnecessary: CPU-bound or instant operations
kessoku.Provide(NewConfig)  // Just creates struct
kessoku.Provide(NewLogger)  // In-memory only
```

### 2. Organize with Sets by Domain

```go
var AuthSet = kessoku.Set(...)
var DatabaseSet = kessoku.Set(...)
var CacheSet = kessoku.Set(...)

var _ = kessoku.Inject[*App](
    "InitializeApp",
    AuthSet,
    DatabaseSet,
    CacheSet,
    kessoku.Provide(NewApp),
)
```

### 3. Bind Interfaces for Testability

```go
// Production
var ProductionSet = kessoku.Set(
    kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
)

// Testing (define in test file)
var TestSet = kessoku.Set(
    kessoku.Bind[Repository](kessoku.Provide(NewMockRepo)),
)
```

### 4. Handle Errors at Provider Level

```go
// Good: Return error from provider
func NewDB(cfg *Config) (*DB, error) {
    if cfg.DSN == "" {
        return nil, errors.New("DSN required")
    }
    return sql.Open("postgres", cfg.DSN)
}

// Bad: Panic in provider
func NewDB(cfg *Config) *DB {
    db, err := sql.Open("postgres", cfg.DSN)
    if err != nil {
        panic(err)  // Don't do this
    }
    return &DB{db: db}
}
```
