# Research: Wire to Kessoku Migration Tool

**Date**: 2026-01-02
**Status**: Complete

## 1. Wire API Analysis

### Source: google/wire v0.7.0

Wire provides the following API functions for dependency injection:

| Function | Signature | Purpose |
|----------|-----------|---------|
| `Build` | `Build(...interface{}) string` | Injector template marker |
| `NewSet` | `NewSet(...interface{}) ProviderSet` | Creates provider set |
| `Bind` | `Bind(iface, to interface{}) Binding` | Interface to implementation binding |
| `Value` | `Value(interface{}) ProvidedValue` | Constant value provider |
| `InterfaceValue` | `InterfaceValue(typ, x interface{}) ProvidedValue` | Interface constant binding |
| `Struct` | `Struct(structType interface{}, fieldNames ...string) StructProvider` | Struct field injection |
| `FieldsOf` | `FieldsOf(structType interface{}, fieldNames ...string) StructFields` | Field extraction provider |

**Note**: Wire is no longer maintained as of v0.7.0 (Aug 2025).

## 2. Kessoku API Analysis

### Source: annotation.go

Kessoku provides these corresponding functions:

| Function | Signature | Purpose |
|----------|-----------|---------|
| `Inject` | `Inject[T any](name, ...provider) struct{}` | Injector declaration |
| `Set` | `Set(...provider) set` | Provider set grouping |
| `Provide` | `Provide[T any](fn T) fnProvider[T]` | Function provider wrapper |
| `Bind` | `Bind[S, T any, F funcProvider[T]](fn F) bindProvider[S, T, F]` | Interface binding with generics |
| `Value` | `Value[T any](v T) fnProvider[func() T]` | Constant value provider |
| `Async` | `Async[T any, F funcProvider[T]](fn F) asyncProvider[T, F]` | Parallel execution wrapper |
| `Struct` | `Struct[T any]() structProvider[T]` | Struct field expansion |

## 3. Transformation Rules

### Decision: Pattern Mapping

| Wire Pattern | Kessoku Pattern | Notes |
|--------------|-----------------|-------|
| `wire.NewSet(...)` | `kessoku.Set(...)` | Direct mapping, wrap providers with `Provide()` |
| `wire.Bind(new(I), new(Impl))` | `kessoku.Bind[I](kessoku.Provide(NewImpl))` | Requires type parameter, constructor lookup |
| `wire.Value(v)` | `kessoku.Value(v)` | Direct mapping |
| `wire.InterfaceValue(new(I), v)` | `kessoku.Bind[I](kessoku.Value(v))` | Expand to Bind + Value |
| `wire.Struct(new(T), "*")` | `kessoku.Provide(func(...) *T {...})` | Generate constructor function |
| `wire.Struct(new(T), "F1", "F2")` | `kessoku.Provide(func(...) *T {...})` | Selective field constructor |
| `wire.FieldsOf(new(T), "F1", "F2")` | `kessoku.Provide(func(t *T) (F1Type, F2Type) {...})` | Multi-value field extractor |
| Provider function `NewFoo` | `kessoku.Provide(NewFoo)` | Wrap in Provide() |

**Rationale**: These mappings preserve semantic equivalence while leveraging kessoku's type-safe generics.

**Alternatives Considered**:
- Direct AST transplant without transformation: Rejected because wire uses reflection-based API while kessoku uses generics
- String-based code generation: Rejected in favor of AST manipulation for type safety

### Decision: Struct Transformation Strategy

For `wire.Struct(new(Config), "*")` where Config has Field1 and Field2:

```go
// Input
wire.Struct(new(Config), "*")

// Output
kessoku.Provide(func(f1 Field1Type, f2 Field2Type) *Config {
    return &Config{Field1: f1, Field2: f2}
})
```

**Rationale**: Kessoku's `Struct[T]()` is for field expansion (providing field values), not struct construction. Wire's `Struct` creates the struct itself.

**Alternatives Considered**:
- Use kessoku.Struct directly: Rejected because semantics differ (kessoku.Struct expands fields, wire.Struct constructs)

### Decision: FieldsOf Transformation Strategy

For `wire.FieldsOf(new(Config), "DB", "Cache")`:

```go
// Input
wire.FieldsOf(new(Config), "DB", "Cache")

// Output (single provider with multiple return values)
kessoku.Provide(func(c *Config) (DBType, CacheType) { return c.DB, c.Cache }),
```

**Rationale**: A single provider with multiple return values is more efficient than multiple providers. Kessoku supports multi-value returns from provider functions.

### Decision: Build/Injector Handling

`wire.Build` is OUT OF SCOPE per spec (FR-010). The migration tool will:
1. Detect `wire.Build` calls
2. Emit warning: "Unsupported pattern: wire.Build at [location]"
3. Continue processing other patterns

**Rationale**: `wire.Build` creates injector functions which require different treatment than provider sets.

## 4. CLI Integration

### Decision: Kong Subcommand Pattern

Extend existing CLI with subcommand structure:

```go
type CLI struct {
    Generate GenerateCmd `cmd:"" default:"1" help:"Generate DI code (default)"`
    Migrate  MigrateCmd  `cmd:"" help:"Migrate wire config to kessoku"`
    Version  VersionFlag `help:"Show version"`
}

type MigrateCmd struct {
    Output string   `short:"o" default:"kessoku.go" help:"Output file path"`
    Files  []string `arg:"" help:"Wire files to migrate"`
}
```

**Rationale**: Kong's subcommand pattern maintains backward compatibility (generate is default) while adding new functionality.

**Alternatives Considered**:
- Separate `kessoku-migrate` binary: Rejected for user convenience
- Flag-based mode switching: Rejected for clarity

## 5. AST Parsing Strategy

### Decision: Use golang.org/x/tools/go/packages

```go
cfg := &packages.Config{
    Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
}
pkgs, _ := packages.Load(cfg, patterns...)
```

**Rationale**: Provides type information needed for:
- Resolving struct field types for Struct/FieldsOf transformation
- Finding constructor functions for Bind transformation
- Package path resolution

**Alternatives Considered**:
- go/parser only: Rejected because type information is essential
- go/types directly: Packages wraps this with file loading

## 6. Output Generation Strategy

### Decision: AST-based Code Generation

1. Parse input files to AST
2. Transform wire patterns to kessoku AST nodes
3. Generate output using go/printer or go/format

**Rationale**: Produces correctly formatted Go code without manual string manipulation.

### Decision: Import Handling

1. Remove `github.com/google/wire` import
2. Add `github.com/mazrean/kessoku` import
3. Merge imports from multiple files, deduplicate

### Decision: Build Tags and Comments (EC-008, EC-009)

**Build Tags**: Wire files typically have `//go:build wireinject` or `// +build wireinject` constraints. These are wire-specific and NOT preserved in output because:
- Wire build tags prevent normal compilation
- Kessoku files need no special build constraints
- Output uses `//go:generate` directive instead

**Comments**: Comments attached to wire function calls are NOT preserved:
- Wire-specific comments (explaining wire.Bind usage, etc.) become irrelevant
- Generated kessoku code is self-explanatory
- Preserving comments would require complex AST manipulation with minimal benefit

**Rationale**: Clean output without wire-specific artifacts.

## 7. Error Handling Strategy

### Decision: Continue on Warning, Stop on Error

| Condition | Action |
|-----------|--------|
| Unsupported pattern (wire.Build) | Warn, continue |
| No wire patterns found | Warn, skip file |
| Syntax error in file | Error, abort |
| Type resolution failure | Error, abort |
| Missing constructor for Bind | Error, abort |
| Package mismatch in merge | Error, abort |
| Same-name collision in merge | Error, abort |

**Rationale**: Allow partial migration while ensuring output correctness. Constructor lookup is mandatory for Bind transformation - if not found, the generated code would be invalid.

## 8. Testing Strategy

### Decision: Table-Driven Golden File Tests

```text
testdata/
├── basic/
│   ├── input.go       # Wire source
│   └── expected.go    # Expected kessoku output
├── struct/
│   ├── input.go
│   └── expected.go
└── ...
```

**Rationale**: Easy to add cases, clear expected behavior.
