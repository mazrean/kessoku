# Implementation Plan: Struct Annotation for Field Expansion

**Branch**: `001-struct-annotation` | **Date**: 2026-01-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-struct-annotation/spec.md`

## Summary

Implement a `kessoku.Struct[T]()` annotation that automatically expands all exported fields of a struct type as individual dependencies. This requires changes to the parser (to extract struct field metadata), graph construction (to create synthetic field accessor nodes and emit field access statements), and code generator (to handle the new statement type). The implementation follows existing kessoku patterns while adding new infrastructure to carry struct field metadata through the pipeline.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: github.com/alecthomas/kong (CLI), golang.org/x/tools/go/packages (AST analysis)
**Storage**: N/A (code generation tool)
**Testing**: go test -v ./...
**Target Platform**: Cross-platform CLI tool (Linux, Windows, macOS)
**Project Type**: Single module with tools submodule
**Performance Goals**: Code generation completes within seconds for typical projects
**Constraints**: Must maintain backwards compatibility with existing kessoku annotations
**Scale/Scope**: Single feature addition to existing codebase

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: The project constitution is currently a template. The following checks are based on standard Go project practices and kessoku's existing patterns:

| Check | Status | Notes |
|-------|--------|-------|
| Follows existing kessoku annotation patterns | PASS | Uses same provider interface pattern as Provide, Async, Bind, Set |
| Maintains backwards compatibility | PASS | New annotation, no changes to existing API |
| Uses existing error handling patterns | REQUIRES CHANGES | Need to add validation before graph construction |
| Follows TDD methodology | PENDING | Tests to be written first per CLAUDE.md guidelines |
| Generates deterministic output | REQUIRES CHANGES | Must implement field ordering per FR-012 |

## Changes Required

### 0. Public API Changes (`annotation.go`)

**FR-001**: Add `kessoku.Struct[T]()` annotation function.

**Current State**: `annotation.go` defines `Provide`, `Async`, `Bind`, `Set`, `Value`, `Arg` functions and their underlying provider types (`fnProvider`, `asyncProvider`, `bindProvider`, etc.).

**Required Changes**:

#### 0.1 Add `structProvider` type

```go
// structProvider marks a struct type for field expansion.
// When used in an Inject declaration, all exported fields of T
// become available as individual dependencies.
type structProvider[T any] struct{}

// provide implements the provider interface.
// This makes structProvider usable in Inject() calls.
func (s structProvider[T]) provide() {}
```

#### 0.2 Add `Struct[T]()` function

```go
// Struct returns a provider that expands all exported fields of T
// as individual dependencies for injection.
//
// T must be a struct type or pointer to struct type.
// All exported fields of T become available as injectable dependencies.
// Unexported fields are ignored.
//
// The struct type T must be provided by another provider in the same
// Inject call (e.g., via Provide). Struct only expands fields; it does
// not create the struct instance.
//
// Example:
//
//	var _ = kessoku.Inject[*App](
//	    "InitializeApp",
//	    kessoku.Provide(NewConfig),     // Provides *Config
//	    kessoku.Struct[*Config](),      // Expands Config fields
//	    kessoku.Provide(NewApp),
//	)
func Struct[T any]() structProvider[T] {
    return structProvider[T]{}
}
```

**Integration**: The `structProvider` type is recognized by the parser (Section 1.1) via its type name. The parser extracts the type parameter `T` from the generic instantiation and creates a `ProviderSpec` with `Type: ProviderTypeStruct`.

### 1. Parser Changes (`internal/kessoku/parser.go`)

**Current State**: `parseProviderType()` returns `(requires, provides, isReturnError, isAsync, error)` - no slot for field metadata.

**Required Changes**:

#### 1.1 Add `structProvider` case to `parseProviderType()` (line ~446)

This case validates the type argument is a struct type and returns an error if not (FR-008, SC-004):

```go
case "structProvider":
    if typeArgs.Len() < 1 {
        return nil, nil, false, false, fmt.Errorf("structProvider requires 1 type argument")
    }
    structType := typeArgs.At(0)

    // FR-008: Validate that T is a struct type
    underlying := structType
    if ptr, ok := structType.(*types.Pointer); ok {
        underlying = ptr.Elem()
    }
    if named, ok := underlying.(*types.Named); ok {
        underlying = named.Underlying()
    }
    if _, ok := underlying.(*types.Struct); !ok {
        // SC-004: Error message MUST contain "not a struct type" and the actual type name
        return nil, nil, false, false, fmt.Errorf("not a struct type: %s", structType)
    }

    // Return struct type as requirement - field extraction happens in parseProviderArgument
    return []types.Type{structType}, nil, false, false, nil
```

#### 1.2 Metadata handoff in `parseProviderArgument()` (line ~347)

Since `parseProviderType()` cannot return field metadata (signature limitation), the metadata is attached AFTER the call. The struct type is obtained from the `requires` return value:

```go
// In parseProviderArgument(), after calling parseProviderType():
providerType := pkg.TypesInfo.TypeOf(arg)
named := providerType.(*types.Named)

// parseProviderType returns: requires, provides, isReturnError, isAsync, err
requires, provides, isReturnError, isAsync, err := p.parseProviderType(named, pkg, imports, varPool)
if err != nil {
    return err
}

// Existing code creates ProviderSpec with the parsed values...
providerSpec := &ProviderSpec{
    ASTExpr:       arg,
    Type:          ProviderTypeFunction, // default
    Requires:      requires,
    Provides:      provides,
    IsReturnError: isReturnError,
    IsAsync:       isAsync,
}
build.Providers = append(build.Providers, providerSpec)

// NEW: For struct providers, attach metadata using the requires[0] as structType
if named.Obj().Name() == "structProvider" {
    // structType is requires[0] - returned by parseProviderType case "structProvider"
    structType := requires[0]
    providerSpec.Type = ProviderTypeStruct
    providerSpec.StructType = structType
    providerSpec.StructFields = p.extractExportedFields(structType, pkg, imports, varPool)
}
```

#### 1.3 New helper function `extractExportedFields()`

**Precondition**: This function is only called after `parseProviderType()` has validated that the type is a struct. The validation in Section 1.1 ensures non-struct types never reach this point.

```go
func (p *Parser) extractExportedFields(structType types.Type, pkg *packages.Package, imports map[string]*Import, varPool *VarPool) []*StructFieldSpec {
    // Get underlying struct (handle pointer types)
    // Note: parseProviderType has already validated this is a struct type
    underlying := structType
    if ptr, ok := structType.(*types.Pointer); ok {
        underlying = ptr.Elem()
    }
    if named, ok := underlying.(*types.Named); ok {
        underlying = named.Underlying()
    }

    structUnderlying := underlying.(*types.Struct) // Safe: validated in parseProviderType

    fields := make([]*StructFieldSpec, 0)
    for i := 0; i < structUnderlying.NumFields(); i++ {
        field := structUnderlying.Field(i)
        // FR-003: Ignore unexported fields (including unexported embedded types)
        if !field.Exported() {
            continue
        }
        fields = append(fields, &StructFieldSpec{
            Name:      field.Name(),
            Type:      field.Type(),
            Index:     i,
            Anonymous: field.Anonymous(), // FR-009: Track embedded fields
        })
    }

    // FR-012: Sort alphabetically by name for deterministic output
    sort.Slice(fields, func(i, j int) bool {
        return fields[i].Name < fields[j].Name
    })

    return fields
}
```

### 2. Provider Data Structure Changes (`internal/kessoku/provider.go`)

**Current State**: `ProviderSpec` has `Provides [][]types.Type` and `Requires []types.Type`.

**Required Changes**:

#### 2.1 New constants

```go
const (
    ProviderTypeFunction    ProviderType = "function"
    ProviderTypeArg         ProviderType = "arg"
    ProviderTypeStruct      ProviderType = "struct"       // NEW
    ProviderTypeFieldAccess ProviderType = "field_access" // NEW
)
```

#### 2.2 New `StructFieldSpec` struct

```go
type StructFieldSpec struct {
    Name      string
    Type      types.Type
    Index     int
    Anonymous bool
}
```

#### 2.3 Extended `ProviderSpec`

```go
type ProviderSpec struct {
    // Existing fields...

    // NEW: For FR-012 deterministic ordering
    DeclOrder int // Declaration order for stable sorting when topological order is equal

    // NEW: For ProviderTypeStruct
    StructType   types.Type
    StructFields []*StructFieldSpec

    // NEW: For ProviderTypeFieldAccess
    SourceField *StructFieldSpec
}
```

#### 2.4 New `InjectorFieldAccessStmt` implementing `InjectorStmt`

```go
type InjectorFieldAccessStmt struct {
    StructParam *InjectorParam
    Field       *StructFieldSpec
    ReturnParam *InjectorParam
}

func (stmt *InjectorFieldAccessStmt) Stmt(varPool *VarPool, injector *Injector, returnErrStmts func(ast.Expr) []ast.Stmt) ([]ast.Stmt, []string) {
    // Generate: fieldVar := structVar.FieldName
    return []ast.Stmt{
        &ast.AssignStmt{
            Lhs: []ast.Expr{ast.NewIdent(stmt.ReturnParam.Name(varPool))},
            Tok: token.DEFINE,
            Rhs: []ast.Expr{
                &ast.SelectorExpr{
                    X:   ast.NewIdent(stmt.StructParam.Name(varPool)),
                    Sel: ast.NewIdent(stmt.Field.Name),
                },
            },
        },
    }, nil
}

func (stmt *InjectorFieldAccessStmt) HasAsync() bool {
    return false
}
```

### 3. Graph Changes (`internal/kessoku/graph.go`)

**Current State**: `NewGraph()` only handles function providers via `fnProviderMap`. `Build()` creates `InjectorProviderCallStmt`. `buildPoolStmts()` emits only `InjectorProviderCallStmt`.

**Required Changes**:

#### 3.1 Struct provider expansion in `NewGraph()` (line ~261)

Two-pass approach to ensure struct providers are processed after function providers.

**FR-012 Ordering Guarantee**:
1. `build.Providers` preserves declaration order from `kessoku.Inject()` (including in-place Set expansion)
2. Struct providers are processed in their declaration order (second pass iterates in order)
3. Each struct's fields are pre-sorted alphabetically in `extractExportedFields()` (Section 1.3)
4. Synthetic field accessors are added to the graph with a `DeclOrder` field tracking their creation sequence
5. `Build()` uses `DeclOrder` as a secondary sort key for providers with the same topological level

```go
fnProviderMap := make(map[string]*fnProvider)
declOrder := 0 // FR-012: Track declaration order for deterministic output

// First pass: register function providers with DeclOrder (existing code modified)
for _, provider := range build.Providers {
    if provider.Type == ProviderTypeStruct {
        continue // Handle in second pass (preserves declaration order)
    }

    // FR-012: Assign DeclOrder to function providers
    provider.DeclOrder = declOrder
    declOrder++

    // ... existing fnProviderMap population
    for groupIndex, typeGroup := range provider.Provides {
        for _, typ := range typeGroup {
            typeKey := typ.String()
            if _, exists := fnProviderMap[typeKey]; exists {
                return nil, fmt.Errorf("multiple providers provide %s", typeKey)
            }
            fnProviderMap[typeKey] = &fnProvider{
                provider:    provider,
                returnIndex: groupIndex,
            }
        }
    }
}

// Second pass: expand struct providers (in declaration order per FR-012)
// Note: ProviderTypeStruct providers are NOT added to the graph as nodes.
// They are markers that create synthetic ProviderTypeFieldAccess providers.
// The struct type itself is provided by a function provider (already in fnProviderMap).
for _, provider := range build.Providers {
    if provider.Type != ProviderTypeStruct {
        continue
    }

    // FR-010: Validate struct provider exists
    structTypeKey := provider.StructType.String()
    if _, ok := fnProviderMap[structTypeKey]; !ok {
        // FR-006: Check for type mismatch
        var altTypeKey string
        if ptr, isPtr := provider.StructType.(*types.Pointer); isPtr {
            altTypeKey = ptr.Elem().String()
        } else {
            altTypeKey = "*" + structTypeKey
        }
        if _, hasAlt := fnProviderMap[altTypeKey]; hasAlt {
            return nil, fmt.Errorf("type mismatch: expected %s, got %s", structTypeKey, altTypeKey)
        }
        return nil, fmt.Errorf("no provider for type %s", structTypeKey)
    }

    // Create synthetic field accessor providers
    // FR-012: Fields already sorted alphabetically, so iterating in order preserves alphabetical ordering
    // FR-012: declOrder++ for each field ensures:
    //   - Fields within a struct are ordered alphabetically (by iteration order)
    //   - Struct A's fields come before Struct B's fields (by declaration order in Inject)
    for _, field := range provider.StructFields {
        fieldTypeKey := field.Type.String()
        if _, exists := fnProviderMap[fieldTypeKey]; exists {
            return nil, fmt.Errorf("multiple providers provide %s", fieldTypeKey)
        }

        syntheticProvider := &ProviderSpec{
            Type:          ProviderTypeFieldAccess,
            Requires:      []types.Type{provider.StructType},
            Provides:      [][]types.Type{{field.Type}},
            SourceField:   field,
            DeclOrder:     declOrder, // FR-012: Track for deterministic ordering
        }
        declOrder++

        fnProviderMap[fieldTypeKey] = &fnProvider{
            provider:    syntheticProvider,
            returnIndex: 0,
        }
    }
}
```

**Note on ProviderSpec extension**: Add `DeclOrder int` field to `ProviderSpec` in Section 2.3.

#### 3.2 Ordering in `Build()` and statement emission in `buildPoolStmts()` (line ~1037)

**FR-012: DeclOrder usage in Build()**:

The `Build()` function performs BFS-based topological sort. To ensure FR-012 deterministic ordering:

1. Each node tracks its `TopoLevel` (distance from root in dependency graph)
2. Nodes are first sorted by `TopoLevel` (primary) to respect dependencies
3. Within the same `TopoLevel`, nodes are sorted by `DeclOrder` (secondary) for determinism

```go
// In node struct, add:
type node struct {
    // ... existing fields
    TopoLevel int // Topological level (distance from root)
}

// In Build(), during BFS, assign TopoLevel:
for !queue.Empty() {
    n := queue.Pop()
    for _, dep := range n.dependencies {
        if dep.TopoLevel <= n.TopoLevel {
            dep.TopoLevel = n.TopoLevel + 1
        }
        // ... existing logic
    }
}

// After BFS completes, stable sort by (TopoLevel, DeclOrder):
sort.SliceStable(pool, func(i, j int) bool {
    // Primary: topological level (dependencies must come first)
    if pool[i].TopoLevel != pool[j].TopoLevel {
        return pool[i].TopoLevel < pool[j].TopoLevel
    }
    // Secondary: DeclOrder for deterministic ordering within same level
    return pool[i].providerSpec.DeclOrder < pool[j].providerSpec.DeclOrder
})
```

**Statement emission**:
Add handling for `ProviderTypeFieldAccess`:

```go
func (g *Graph) buildPoolStmts(pool []*node, ...) ([]InjectorStmt, error) {
    stmts := make([]InjectorStmt, 0, len(pool))

    for _, n := range pool {
        if n.providerSpec == nil {
            continue
        }

        // NEW: Handle field access providers differently
        if n.providerSpec.Type == ProviderTypeFieldAccess {
            stmts = append(stmts, &InjectorFieldAccessStmt{
                StructParam: n.providerArgs[0].Param, // The struct dependency
                Field:       n.providerSpec.SourceField,
                ReturnParam: n.returnValues[0],
            })
            continue
        }

        // Existing code for InjectorProviderCallStmt
        stmts = append(stmts, &InjectorProviderCallStmt{...})
    }
    return stmts, nil
}
```

### 4. Generator Changes (`internal/kessoku/generator.go`)

**Current State**: Handles `InjectorProviderCallStmt` and `InjectorChainStmt` via `InjectorStmt` interface.

**Required Changes**: None to generator.go itself - the `InjectorStmt` interface handles polymorphism. The `Stmt()` method on `InjectorFieldAccessStmt` (defined in provider.go) generates the correct AST.

### 5. FR-007: Integration with Existing Annotations

**Provide**: Struct providers depend on function providers. The struct type must be provided by `kessoku.Provide()` before `kessoku.Struct()` can expand its fields. This is enforced by the two-pass approach in `NewGraph()`.

**Async**: When struct is provided by async provider:
- Dependency graph naturally handles this
- Field accessor nodes depend on struct node
- Existing `IsWait` mechanism ensures field extraction waits for async completion
- No special handling needed

**Bind**: Field types can be bound to interfaces:
- After `Struct` expands fields, the field type is in `fnProviderMap`
- `Bind` can then map concrete field type to interface
- Example: `Struct[*Config]()` provides `*FileLogger`, then `Bind[Logger](...)` maps it

**Set**: Struct can be used inside Set:
- `parseProviderArgument()` recursively processes Set arguments
- `Struct` inside Set is expanded at Set's position
- Ordering preserved: Set arguments processed in order

### 6. Error Message Alignment (SC-004)

| Error | Message | Location |
|-------|---------|----------|
| Non-struct type | "not a struct type: `<type>`" | `parseProviderType()` |
| Missing provider | "no provider for type `<type>`" | `NewGraph()` |
| Type mismatch | "type mismatch: expected `<expected>`, got `<actual>`" | `NewGraph()` |
| Duplicate types | "multiple providers provide `<type>`" | `NewGraph()` (existing) |

### 7. Test Requirements

#### 7.1 Unit Tests (`*_test.go`)

| File | Test Cases |
|------|------------|
| `parser_test.go` | Struct field extraction, pointer/value types, unexported field filtering, embedded field handling, alphabetical ordering |
| `graph_test.go` | Struct expansion, missing provider error, type mismatch error, duplicate field type error |
| `generator_test.go` | `InjectorFieldAccessStmt.Stmt()` AST generation |

#### 7.2 Integration Tests (`processor_test.go`)

| Scenario | Tests FR |
|----------|----------|
| Basic field expansion | FR-001, FR-002, FR-004 |
| Unexported fields ignored | FR-003 |
| Pointer/value type mismatch | FR-006 |
| Struct + Provide integration | FR-007 |
| Struct + Async integration | FR-007, User Story 3.4 |
| Struct + Bind integration | FR-007, User Story 3.5 |
| Struct + Set integration | FR-007, User Story 3.2, 3.6 |
| Non-struct type error | FR-008 |
| Embedded fields | FR-009 |
| Missing provider error | FR-010 |
| No exported fields | FR-011 |
| Field ordering (alphabetical) | FR-012 |
| Multiple Struct ordering | FR-012 |
| Set in-place expansion | FR-012 |

#### 7.3 Success Criteria Test Assertions

**SC-001: Provider Count and Type Verification**

For each test case with N exported fields, assert both count AND types match:
```go
// In graph_test.go
func TestStructProviderCountAndTypes(t *testing.T) {
    // Given: struct Config with 3 exported fields:
    //   - Apple string
    //   - Mango bool
    //   - Zebra int
    graph, err := NewGraph(build)
    require.NoError(t, err)

    // Collect field accessor providers
    var fieldAccessors []*ProviderSpec
    for _, provider := range graph.providers {
        if provider.Type == ProviderTypeFieldAccess {
            fieldAccessors = append(fieldAccessors, provider)
        }
    }

    // SC-001: Assert count matches number of exported fields
    assert.Equal(t, 3, len(fieldAccessors), "SC-001: must generate exactly N field providers")

    // SC-001: Assert types match the struct's exported field types
    expectedTypes := map[string]bool{
        "string": false, // Apple
        "bool":   false, // Mango
        "int":    false, // Zebra
    }
    for _, fa := range fieldAccessors {
        providedType := fa.Provides[0][0].String()
        _, expected := expectedTypes[providedType]
        assert.True(t, expected, "SC-001: unexpected field type %s", providedType)
        expectedTypes[providedType] = true
    }
    for typeName, found := range expectedTypes {
        assert.True(t, found, "SC-001: expected field type %s not provided", typeName)
    }
}

func TestStructProviderWithNoExportedFields(t *testing.T) {
    // Given: struct with only unexported fields
    graph, err := NewGraph(build)
    require.NoError(t, err)

    // SC-001 + FR-011: No field providers generated (count = 0)
    fieldAccessCount := 0
    for _, provider := range graph.providers {
        if provider.Type == ProviderTypeFieldAccess {
            fieldAccessCount++
        }
    }
    assert.Equal(t, 0, fieldAccessCount, "SC-001/FR-011: no providers for struct with no exported fields")
}
```

**SC-003: Compile Sanity Check**

Integration tests verify generated code compiles:
```go
// In processor_test.go
func TestGeneratedCodeCompiles(t *testing.T) {
    // Given: valid Struct annotation
    err := processor.Process(inputFile)
    require.NoError(t, err)

    // Then: generated file exists and compiles
    cmd := exec.Command("go", "build", "./...")
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err, "SC-003: generated code must compile without errors: %s", output)
}
```

**SC-004: Error Message String Assertions**

Each error case tests for required phrases:
```go
// In parser_test.go
func TestNonStructTypeError(t *testing.T) {
    // Given: kessoku.Struct[string]()
    _, err := parser.Parse(input)

    // SC-004: must contain "not a struct type" and the type name
    require.Error(t, err)
    assert.Contains(t, err.Error(), "not a struct type", "SC-004: error must contain 'not a struct type'")
    assert.Contains(t, err.Error(), "string", "SC-004: error must contain actual type name")
}

// In graph_test.go
func TestMissingProviderError(t *testing.T) {
    // Given: kessoku.Struct[*Config]() without Provide(NewConfig)
    _, err := NewGraph(build)

    // SC-004: must contain "no provider" and the type name
    require.Error(t, err)
    assert.Contains(t, err.Error(), "no provider", "SC-004: error must contain 'no provider'")
    assert.Contains(t, err.Error(), "*Config", "SC-004: error must contain missing type name")
}

func TestTypeMismatchError(t *testing.T) {
    // Given: kessoku.Struct[*Config]() with Provide(NewConfig) returning Config (not *Config)
    _, err := NewGraph(build)

    // SC-004: must contain "type mismatch" and both types
    require.Error(t, err)
    assert.Contains(t, err.Error(), "type mismatch", "SC-004: error must contain 'type mismatch'")
    assert.Contains(t, err.Error(), "*Config", "SC-004: error must contain expected type")
    assert.Contains(t, err.Error(), "Config", "SC-004: error must contain actual type")
}

func TestDuplicateProviderError(t *testing.T) {
    // Given: struct with two string fields (type conflict)
    _, err := NewGraph(build)

    // SC-004: must contain "multiple providers provide" and the type
    require.Error(t, err)
    assert.Contains(t, err.Error(), "multiple providers provide", "SC-004: error must contain 'multiple providers provide'")
    assert.Contains(t, err.Error(), "string", "SC-004: error must contain conflicting type name")
}
```

#### 7.4 User Story Acceptance Tests

**User Story 1: Expand Config Struct Fields** (`processor_test.go`)

| Scenario | Test Function | Assertion |
|----------|---------------|-----------|
| 1.1 Basic field expansion | `TestStructBasicFieldExpansion` | Struct with `DBHost string` and `DBPort int` → both types become injectable dependencies |
| 1.2 Exported fields only | `TestStructExportedFieldsOnly` | Struct with mixed fields → only exported fields in provider list |
| 1.3 Unexported ignored | `TestStructUnexportedIgnored` | Struct with `host string` (unexported) → no provider for `string` |
| 1.4 Alphabetical order | `TestStructFieldAlphabeticalOrder` | Struct `{Zebra int, Apple string, Mango bool}` → generated order: Apple, Mango, Zebra |
| 1.5 Multiple struct order | `TestStructMultipleStructOrdering` | `Struct[*ConfigA]()` then `Struct[*ConfigB]()` → ConfigA fields before ConfigB fields |

**User Story 2: Type-Safe Field Access** (`processor_test.go`)

| Scenario | Test Function | Assertion |
|----------|---------------|-----------|
| 2.1 Multiple types | `TestStructDifferentFieldTypes` | Struct with string, int, custom type → each accessor returns correct type |
| 2.2 Pointer/value fields | `TestStructPointerAndValueFields` | Struct with `*Logger` and `Config` → both handled correctly, types preserved |

**User Story 3: Use with Existing Annotations** (`processor_test.go`)

| Scenario | Test Function | Assertion |
|----------|---------------|-----------|
| 3.1 Provide + Struct | `TestStructWithProvide` | `Provide(NewConfig)` + `Struct[*Config]()` → Config created first, then fields extracted |
| 3.2 Set with Struct | `TestStructInsideSet` | Set containing `Struct` → fields properly expanded at Set position |
| 3.3 Missing provider | `TestStructMissingProvider` | `Struct[*Config]()` without provider → error contains "no provider" and "*Config" |
| 3.4 Async + Struct | `TestStructWithAsync` | `Async(Provide(NewConfig))` + `Struct[*Config]()` → field extraction waits for async |
| 3.5 Bind + Struct | `TestStructWithBind` | Field type `DB` + `Bind[Database]` → `DB` field can satisfy `Database` dependency |
| 3.6 Set ordering | `TestStructSetInPlaceOrdering` | Set with `Struct[*A]()`, `Struct[*B]()` between providers → A fields, B fields at Set position |

**User Story 4: Handle Embedded Fields** (`processor_test.go`)

| Scenario | Test Function | Assertion |
|----------|---------------|-----------|
| 4.1 Embedded value | `TestStructEmbeddedValue` | `type App struct { Config }` → `Config` value becomes dependency |
| 4.2 Embedded pointer | `TestStructEmbeddedPointer` | `type App struct { *Config }` → `*Config` pointer becomes dependency |
| 4.3 No recursive expansion | `TestStructNoRecursiveExpansion` | Embedded struct's fields NOT expanded (only direct fields) |
| 4.4 Unexported embedded | `TestStructUnexportedEmbedded` | `type App struct { config }` (unexported) → no dependency produced |

## Project Structure

### Documentation (this feature)

```text
specs/001-struct-annotation/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # N/A for CLI tool
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Changes marked with + (add) or ~ (modify)
├── annotation.go              # + Add Struct[T]() function and structProvider type
├── cmd/kessoku/              # No changes
├── internal/
│   ├── config/               # No changes
│   ├── kessoku/
│   │   ├── parser.go         # ~ Add structProvider case, extractExportedFields()
│   │   ├── graph.go          # ~ Add struct expansion in NewGraph(), field access in buildPoolStmts()
│   │   ├── generator.go      # No changes (InjectorStmt interface handles polymorphism)
│   │   ├── processor.go      # No changes
│   │   ├── provider.go       # ~ Add ProviderTypeStruct, StructFieldSpec, InjectorFieldAccessStmt
│   │   └── const.go          # No changes
│   └── pkg/
│       ├── collection/       # No changes
│       └── strings/          # No changes
├── examples/
│   └── struct_expansion/     # + New example for Struct annotation
└── tests (embedded in *_test.go files)
```

## Complexity Tracking

No constitution violations requiring justification. The implementation requires more changes than initially estimated but follows existing patterns.
