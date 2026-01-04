# Kessoku Troubleshooting Guide

Common issues and solutions when using kessoku.

## Code Generation Issues

### Generated file not created

**Symptom**: No `*_band.go` file after running `go generate`

**Causes & Solutions**:

1. **Missing go:generate directive**
   ```go
   // Add this before your kessoku.Inject declarations
   //go:generate go tool kessoku $GOFILE
   ```

2. **Kessoku not installed**
   ```bash
   go get -tool github.com/mazrean/kessoku/cmd/kessoku
   ```

3. **Wrong file path in directive**
   ```go
   // Correct - uses current file
   //go:generate go tool kessoku $GOFILE

   // Also correct - explicit file
   //go:generate go tool kessoku kessoku.go
   ```

### Generated file outdated

**Symptom**: Changes to providers not reflected

**Solution**: Re-run code generation
```bash
go generate ./...
```

### "undefined: kessoku" error

**Symptom**: `undefined: kessoku` in source file

**Solution**: Add import
```go
import "github.com/mazrean/kessoku"
```

## Dependency Resolution Errors

### Circular dependency detected

**Symptom**: `cycle detected in dependency graph`

**Cause**: Provider A needs B, and B needs A

**Solutions**:

1. **Use interfaces to break the cycle**
   ```go
   // Bad: Direct dependency cycle
   func NewA(b *B) *A { ... }
   func NewB(a *A) *B { ... }

   // Good: Interface breaks cycle
   type BInterface interface { DoSomething() }
   func NewA(b BInterface) *A { ... }
   func NewB() *B { ... }
   ```

2. **Restructure dependencies**
   ```go
   // Extract shared dependency
   func NewShared() *Shared { ... }
   func NewA(s *Shared) *A { ... }
   func NewB(s *Shared) *B { ... }
   ```

### No provider for type

**Symptom**: `no provider found for type X`

**Causes & Solutions**:

1. **Missing provider**
   ```go
   var _ = kessoku.Inject[*App](
       "InitializeApp",
       // kessoku.Provide(NewConfig), // Missing!
       kessoku.Provide(NewDB),  // Needs *Config
       kessoku.Provide(NewApp),
   )

   // Fix: Add missing provider
   var _ = kessoku.Inject[*App](
       "InitializeApp",
       kessoku.Provide(NewConfig),  // Added
       kessoku.Provide(NewDB),
       kessoku.Provide(NewApp),
   )
   ```

2. **Type mismatch**
   ```go
   // Provider returns *Config, but dependency needs Config (no pointer)
   func NewConfig() *Config { ... }
   func NewDB(cfg Config) *DB { ... }  // Expects Config, not *Config

   // Fix: Match types
   func NewDB(cfg *Config) *DB { ... }
   ```

3. **Provider in wrong package**
   ```go
   // If NewConfig is in another package, import it
   import "myapp/config"

   var _ = kessoku.Inject[*App](
       "InitializeApp",
       kessoku.Provide(config.NewConfig),
       kessoku.Provide(NewApp),
   )
   ```

### Multiple providers for same type

**Symptom**: `multiple providers for type X`

**Cause**: Two providers return the same type

**Solutions**:

1. **Use different types**
   ```go
   // Bad: Both return string
   kessoku.Value("host"),
   kessoku.Value("port"),

   // Good: Use distinct types
   type DBHost string
   type DBPort string
   kessoku.Value(DBHost("localhost")),
   kessoku.Value(DBPort("5432")),
   ```

2. **Combine into struct**
   ```go
   type DBConfig struct {
       Host string
       Port string
   }
   kessoku.Value(DBConfig{Host: "localhost", Port: "5432"}),
   ```

## Async Issues

### Context not passed to async provider

**Symptom**: `context.Context` parameter is `nil` or missing

**Cause**: Provider signature doesn't accept context

**Solution**: Add context parameter
```go
// Before
func NewDB() (*DB, error) {
    return connectDB()
}

// After
func NewDB(ctx context.Context) (*DB, error) {
    return connectDBWithContext(ctx)
}
```

### Async provider not running in parallel

**Symptom**: Providers run sequentially despite `Async`

**Cause**: Dependency chain prevents parallelization

**Example**:
```go
kessoku.Async(kessoku.Provide(NewConfig)),
kessoku.Async(kessoku.Provide(NewDB)),     // Depends on Config
kessoku.Async(kessoku.Provide(NewCache)),  // Depends on Config
```

Config must complete before DB and Cache can start. DB and Cache will run in parallel after Config completes.

### Runtime panic with Async

**Symptom**: Panic during parallel initialization

**Common Causes**:

1. **Shared mutable state**
   ```go
   // Bad: Shared variable accessed by parallel providers
   var globalConfig *Config

   // Good: Pass as dependency
   func NewDB(cfg *Config) *DB { ... }
   ```

2. **Race condition in provider**
   ```go
   // Ensure providers are thread-safe
   func NewCache(ctx context.Context) (*Cache, error) {
       // Use proper synchronization if needed
   }
   ```

## Interface Binding Issues

### Bind not working

**Symptom**: Dependency receives concrete type instead of interface

**Cause**: Wrong Bind syntax

```go
// Wrong: Bind takes a provider, not just type parameter
kessoku.Bind[Repository](),

// Correct
kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
```

### Interface not satisfied

**Symptom**: `*ConcreteType does not implement Interface`

**Solution**: Ensure implementation has all required methods
```go
type Repository interface {
    Find(id int) (*Entity, error)
    Save(e *Entity) error
}

// Ensure PostgresRepo implements all methods
type PostgresRepo struct { ... }
func (r *PostgresRepo) Find(id int) (*Entity, error) { ... }
func (r *PostgresRepo) Save(e *Entity) error { ... }
```

## Struct Expansion Issues

### Wrong Struct syntax

**Symptom**: Compilation error or unexpected behavior

**Common Mistake**:
```go
// Wrong: Struct doesn't take field arguments
kessoku.Struct[*Config]("Host", "Port"),

// Correct: No arguments, expands ALL exported fields
kessoku.Struct[*Config](),
```

### Fields not injected

**Symptom**: Provider doesn't receive struct fields

**Causes**:

1. **Unexported fields**
   ```go
   type Config struct {
       host string  // unexported - not expanded
       Port int     // exported - expanded
   }
   ```

2. **Type mismatch**
   ```go
   type Config struct {
       Port int  // int
   }
   func NewDB(port string) *DB  // Expects string, not int
   ```

## Runtime Errors

### Nil pointer in generated code

**Symptom**: Panic at runtime with nil pointer

**Causes**:

1. **Provider returns nil without error**
   ```go
   // Bad
   func NewDB() *DB {
       if somethingWrong {
           return nil  // Returns nil without error
       }
       return &DB{}
   }

   // Good
   func NewDB() (*DB, error) {
       if somethingWrong {
           return nil, errors.New("something wrong")
       }
       return &DB{}, nil
   }
   ```

2. **Dependency not initialized**
   - Check all providers are included in Inject

### Cleanup not called

**Symptom**: Resources not cleaned up

**Solution**: Ensure cleanup function is called
```go
app, cleanup, err := InitializeApp(ctx)
if err != nil {
    log.Fatal(err)
}
defer cleanup()  // Must call cleanup
```

## Debugging Tips

### View generated code

Check the `*_band.go` file to understand:
- Provider execution order
- Parallel execution pools
- Error handling flow

### Add logging to providers

```go
func NewDB(ctx context.Context) (*DB, error) {
    log.Println("NewDB starting")
    db, err := connect(ctx)
    log.Println("NewDB completed")
    return db, err
}
```

### Check dependency graph

Manually trace dependencies:
1. List all providers and their dependencies
2. Check for cycles
3. Verify all types are provided
