# Research: Struct Annotation for Field Expansion

**Feature Branch**: `001-struct-annotation`
**Date**: 2026-01-01

## Research Overview

This document captures technical research and design decisions for implementing `kessoku.Struct[T]()`. Special attention is given to how struct metadata flows through the existing parser → graph → generator pipeline.

---

## 1. Current Pipeline Analysis

### 1.1 Existing Data Flow

```
parseProviderType() → ProviderSpec → NewGraph() → node → Build() → InjectorStmt → Generate()
      │                    │              │         │        │            │
      │                    │              │         │        │            │
  returns:              stores:       creates:  stores:   emits:      outputs:
  - requires []Type     - Provides    - fnProviderMap     - InjectorProviderCallStmt
  - provides [][]Type   - Requires    - nodes             - InjectorChainStmt
  - isReturnError       - IsAsync     - edges
  - isAsync             - ASTExpr
```

### 1.2 Key Limitation

The current `parseProviderType()` signature returns only type information:
```go
func (p *Parser) parseProviderType(pkg *packages.Package, providerType types.Type, varPool *VarPool) ([]types.Type, [][]types.Type, bool, bool, error)
```

There's no slot for field names or struct-specific metadata. This requires extending `ProviderSpec` rather than the function signature.

---

## 2. Struct Metadata Flow Design

### 2.1 Parser Stage

**Input**: `kessoku.Struct[*Config]()`

**Processing**:
```go
case "structProvider":
    if typeArgs.Len() < 1 {
        return nil, nil, false, false, fmt.Errorf("structProvider requires 1 type argument")
    }

    structType := typeArgs.At(0)

    // Get underlying struct (handle pointer types)
    underlying := structType
    if ptr, ok := structType.Underlying().(*types.Pointer); ok {
        underlying = ptr.Elem()
    }

    structUnderlying, ok := underlying.Underlying().(*types.Struct)
    if !ok {
        return nil, nil, false, false, fmt.Errorf("not a struct type: %s", structType)
    }

    // Extract and sort fields
    fields := extractExportedFields(structUnderlying) // includes Index for code generation
    sort.Slice(fields, func(i, j int) bool {
        return fields[i].Name < fields[j].Name
    })

    // Return: requires struct type, provides nothing (expanded in graph)
    return []types.Type{structType}, nil, false, false, nil
```

**Key Detail**: Field index is preserved for correct code generation (`config.DBHost` vs `config.fields[0]`).

**Output to ProviderSpec**:
```go
ProviderSpec{
    Type:         ProviderTypeStruct,
    Requires:     []types.Type{*Config},
    Provides:     nil,  // Expanded in graph
    StructType:   *Config,
    StructFields: []*StructFieldSpec{
        {Name: "DBHost", Type: string, Index: 0, Anonymous: false},
        {Name: "DBPort", Type: int, Index: 1, Anonymous: false},
    },
}
```

### 2.2 Graph Stage

**Current `NewGraph()` logic** (simplified):
```go
fnProviderMap := make(map[string]*fnProvider)
for _, provider := range build.Providers {
    for groupIndex, typeGroup := range provider.Provides {
        // ... register providers by type
    }
}
```

**New logic for struct providers**:
```go
// First pass: register function providers
for _, provider := range build.Providers {
    if provider.Type == ProviderTypeStruct {
        continue // Handle in second pass
    }
    // ... existing logic
}

// Second pass: expand struct providers
for _, provider := range build.Providers {
    if provider.Type != ProviderTypeStruct {
        continue
    }

    // FR-010: Check struct provider exists
    structTypeKey := provider.StructType.String()
    if _, ok := fnProviderMap[structTypeKey]; !ok {
        return nil, fmt.Errorf("no provider for type %s", structTypeKey)
    }

    // FR-006: Validate exact type match (already exact by design)

    // Create synthetic field accessor providers
    for _, field := range provider.StructFields {
        fieldTypeKey := field.Type.String()

        // Check for duplicates (FR-005)
        if existing, ok := fnProviderMap[fieldTypeKey]; ok {
            return nil, fmt.Errorf("multiple providers provide %s", fieldTypeKey)
        }

        // Create synthetic provider for field
        syntheticProvider := &ProviderSpec{
            Type:            ProviderTypeFieldAccess,
            Requires:        []types.Type{provider.StructType},
            Provides:        [][]types.Type{{field.Type}},
            SourceField:     field,
            IsReturnError:   false,
            IsAsync:         false,
        }

        fnProviderMap[fieldTypeKey] = &fnProvider{
            provider:    syntheticProvider,
            returnIndex: 0,
        }
    }
}
```

### 2.3 Generator Stage

**New `InjectorFieldAccessStmt`**:
```go
type InjectorFieldAccessStmt struct {
    StructParam *InjectorParam     // The struct instance
    Field       *StructFieldSpec   // Field to extract
    ReturnParam *InjectorParam     // The result
}

func (stmt *InjectorFieldAccessStmt) Stmt(varPool *VarPool, injector *Injector, returnErrStmts func(ast.Expr) []ast.Stmt) ([]ast.Stmt, []string) {
    // Generate: fieldVar := structVar.FieldName
    return []ast.Stmt{
        &ast.AssignStmt{
            Lhs: []ast.Expr{ast.NewIdent(stmt.ReturnParam.Name(varPool))},
            Tok: token.DEFINE,  // or ASSIGN if in async context
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
    return false // Field access is always synchronous
}
```

---

## 3. Type Validation Strategy

### 3.1 Non-Struct Type Detection (FR-008)

**Location**: `parser.go` in `parseProviderType()`

**Logic**:
```go
// Handle pointer to struct
underlying := structType
if ptr, ok := structType.Underlying().(*types.Pointer); ok {
    underlying = ptr.Elem()
}

// Check if it's actually a struct
if _, ok := underlying.Underlying().(*types.Struct); !ok {
    return nil, nil, false, false, fmt.Errorf("not a struct type: %s", structType)
}
```

### 3.2 Missing Provider Detection (FR-010)

**Location**: `graph.go` in `NewGraph()`

**Logic**:
```go
structTypeKey := provider.StructType.String()
if _, ok := fnProviderMap[structTypeKey]; !ok {
    return nil, fmt.Errorf("no provider for type %s", structTypeKey)
}
```

### 3.3 Type Mismatch Detection (FR-006)

**Location**: `graph.go` in `NewGraph()`

The type system handles this naturally:
- `Struct[*Config]` sets `StructType = *Config`
- The lookup in `fnProviderMap` uses the exact type key
- If only `Config` (non-pointer) exists, lookup fails with "no provider" error

To provide a more specific error message:
```go
structTypeKey := provider.StructType.String()
if _, ok := fnProviderMap[structTypeKey]; !ok {
    // Check if opposite pointer/value exists
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
```

---

## 4. Field Ordering Implementation (FR-012)

### 4.1 Within a Struct

Fields are sorted alphabetically when extracted:
```go
func extractExportedFields(structType *types.Struct) []*StructFieldSpec {
    fields := make([]*StructFieldSpec, 0)
    for i := 0; i < structType.NumFields(); i++ {
        field := structType.Field(i)
        if !field.Exported() {
            continue
        }
        fields = append(fields, &StructFieldSpec{
            Name:      field.Name(),
            Type:      field.Type(),
            Index:     i,  // Preserve original index for code generation
            Anonymous: field.Anonymous(),
        })
    }

    // Sort by name
    sort.Slice(fields, func(i, j int) bool {
        return fields[i].Name < fields[j].Name
    })

    return fields
}
```

### 4.2 Multiple Struct Annotations

Structs are processed in declaration order within `Inject`:
```go
kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Struct[*ConfigA](),  // ConfigA fields first (alphabetically)
    kessoku.Struct[*ConfigB](),  // ConfigB fields second (alphabetically)
)
```

This is naturally preserved because `build.Providers` maintains declaration order.

### 4.3 Sets

Sets expand in-place via recursive `parseProviderArgument()`, which preserves order:
```go
// In parseProviderArgument
for _, setArg := range callExpr.Args {
    if err := p.parseProviderArgument(pkg, kessokuPackageScope, setArg, build, imports, fileImports, varPool); err != nil {
        return fmt.Errorf("parse Set provider argument: %w", err)
    }
}
```

---

## 5. Embedded Field Handling (FR-009)

### 5.1 Detection

```go
field := structType.Field(i)
if field.Anonymous() {
    // This is an embedded field
}
```

### 5.2 Behavior

- Embedded fields are treated as regular fields
- The embedded type's value becomes a dependency
- Example: `type App struct { Config }` → provides `Config`
- Example: `type App struct { *Config }` → provides `*Config`
- Nested fields are NOT recursively expanded

### 5.3 Unexported Embedded Types (FR-003)

```go
type App struct {
    config  // unexported embedded type - IGNORED
}
```

The `field.Exported()` check handles this:
```go
if !field.Exported() {
    continue // Skip unexported fields including embedded
}
```

---

## 6. Import Handling for Field Types

### 6.1 During Field Extraction

```go
for _, field := range fields {
    collectImportsFromType(field.Type, pkg, imports, referencedImports, varPool)
}
```

The existing `collectImportsFromType()` function recursively collects imports from types.

### 6.2 In Generated Code

When generating field access:
```go
// For a field of type log.Logger from external package
dbHost := config.Logger  // log package must be imported

// The import is marked as used when the field accessor statement is generated
for _, imp := range stmt.ReturnParam.ReferencedImports {
    imp.IsUsed = true
}
```

---

## 7. Integration with Async Providers

### 7.1 Scenario

```go
kessoku.Async(kessoku.Provide(NewConfig)),
kessoku.Struct[*Config](),
```

### 7.2 Behavior

The dependency graph naturally handles this:
1. Field accessor depends on struct type
2. Struct type is provided by async provider
3. Field accessor waits for async provider via channel

No special handling needed - the existing `IsWait` mechanism works.

---

## 8. Error Path Summary

| Error Condition | Detection Location | Error Message |
|----------------|-------------------|---------------|
| Non-struct type | `parser.go:parseProviderType` | "not a struct type: `<type>`" |
| Missing struct provider | `graph.go:NewGraph` | "no provider for type `<type>`" |
| Type mismatch (ptr/value) | `graph.go:NewGraph` | "type mismatch: expected `<expected>`, got `<actual>`" |
| Duplicate field types | `graph.go:NewGraph` | "multiple providers provide `<type>`" |

---

## 9. Testing Strategy

### 9.1 Parser Tests

```go
func TestParseStructProvider(t *testing.T) {
    // Test cases:
    // - Basic struct field extraction
    // - Pointer to struct
    // - Non-struct type error
    // - Unexported field filtering
    // - Embedded field handling
    // - Field alphabetical ordering
}
```

### 9.2 Graph Tests

```go
func TestStructProviderExpansion(t *testing.T) {
    // Test cases:
    // - Struct fields become providers
    // - Missing struct provider error
    // - Type mismatch error
    // - Duplicate field type error
    // - Ordering of multiple structs
}
```

### 9.3 Generator Tests

```go
func TestFieldAccessGeneration(t *testing.T) {
    // Test cases:
    // - Simple field access statement
    // - Field from external package (imports)
    // - Integration with async context
}
```

---

## 10. Summary of Key Decisions

| Topic | Decision |
|-------|----------|
| Metadata storage | Extend `ProviderSpec` with `StructType`, `StructFields` |
| Graph expansion | Create synthetic providers in `NewGraph()` second pass |
| Field ordering | Sort by name at extraction, preserve declaration order for structs |
| Type validation | Early in parser (non-struct), early in graph (missing/mismatch) |
| Error messages | Match existing style: "multiple providers provide %s" |
| Async handling | Use existing dependency graph semantics |
| Generator | New `InjectorFieldAccessStmt` implementing `InjectorStmt` |
| Imports | Use existing `collectImportsFromType()` mechanism |
