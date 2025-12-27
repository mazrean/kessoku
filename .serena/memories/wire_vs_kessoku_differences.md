# Google Wire vs Kessoku API Differences

## Key Migration Patterns

### 1. Build Declaration
**Wire:**
```go
//go:build wireinject
// +build wireinject

func InitializeApp() *App {
    wire.Build(NewDatabase, NewUserService, NewApp)
    return &App{} // placeholder
}
```

**Kessoku:**
```go
//go:generate go tool kessoku $GOFILE

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewDatabase),
    kessoku.Provide(NewUserService), 
    kessoku.Provide(NewApp),
)
```

### 2. Provider Sets
**Wire:**
```go
var DatabaseSet = wire.NewSet(
    NewConfig,
    NewDatabase,
    NewMigrator,
)
```

**Kessoku:**
```go
var DatabaseSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDatabase),
    kessoku.Provide(NewMigrator),
)
```

### 3. Interface Binding
**Wire:**
```go
var MySet = wire.NewSet(
    wire.Struct(new(MyFoo)),
    wire.Bind(new(Fooer), new(MyFoo))
)
```

**Kessoku:**
```go
var MySet = kessoku.Set(
    kessoku.Bind[Fooer](kessoku.Provide(NewMyFoo)),
)
```

### 4. Value Injection
**Wire:**
```go
wire.Value([]string{"example"})
```

**Kessoku:**
```go
kessoku.Value([]string{"example"})
```

### 5. File Structure Changes
**Wire:**
- Uses `//go:build wireinject` constraint
- Injector functions with placeholder returns
- Generated `wire_gen.go` files

**Kessoku:**
- Uses `//go:generate go tool kessoku $GOFILE`
- Variable declarations with kessoku.Inject
- Generated `*_band.go` files

## Key Transformation Rules
1. Remove `//go:build wireinject` constraints
2. Add `//go:generate go tool kessoku $GOFILE`
3. Convert `wire.Build()` calls to `kessoku.Inject[]` variables
4. Wrap all providers with `kessoku.Provide()`
5. Convert `wire.NewSet()` to `kessoku.Set()`
6. Convert `wire.Bind()` to `kessoku.Bind[]()` with type parameters
7. Keep `wire.Value()` as `kessoku.Value()` (same syntax)
8. Remove injector function bodies (placeholder returns)