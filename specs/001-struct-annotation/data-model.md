# Data Model: Struct Annotation for Field Expansion

**Feature Branch**: `001-struct-annotation`
**Date**: 2026-01-01

## Overview

This document defines the data structures and their integration points for implementing `kessoku.Struct[T]()`. It shows how new types fit into the existing `parser → graph → generator` pipeline.

---

## 1. Public API Types

### 1.1 structProvider

**Location**: `annotation.go`

```go
// structProvider marks a struct type for field expansion.
// When used in an Inject declaration, all exported fields of T
// become available as individual dependencies.
type structProvider[T any] struct{}

// provide implements the provider interface.
func (s structProvider[T]) provide() {}
```

**Constraints**:
- `T` must be a struct type or pointer to struct type
- Validated at code generation time (not compile time)

---

## 2. Internal Types

### 2.1 ProviderType Extension

**Location**: `internal/kessoku/provider.go`

```go
const (
    ProviderTypeFunction    ProviderType = "function"
    ProviderTypeArg         ProviderType = "arg"
    ProviderTypeStruct      ProviderType = "struct"       // NEW: marks struct for expansion
    ProviderTypeFieldAccess ProviderType = "field_access" // NEW: synthetic field accessor
)
```

### 2.2 StructFieldSpec

**Location**: `internal/kessoku/provider.go` (NEW)

```go
// StructFieldSpec represents a field extracted from a struct for dependency injection.
type StructFieldSpec struct {
    Name      string      // Field name (e.g., "DBHost") - used in generated code
    Type      types.Type  // Field type (e.g., string) - used for dependency matching
    Index     int         // Original field index in struct - preserved for proper access
    Anonymous bool        // True for embedded fields - affects naming
}
```

**Why Index is Preserved**:
- Fields are sorted alphabetically for deterministic output
- But code generation needs the original field name, not index
- Example: `config.DBHost` not `config.fields[0]`

### 2.3 Extended ProviderSpec

**Location**: `internal/kessoku/provider.go`

```go
type ProviderSpec struct {
    // Existing fields
    ASTExpr           ast.Expr
    ReferencedImports map[string]*Import
    Type              ProviderType
    Provides          [][]types.Type
    Requires          []types.Type
    IsReturnError     bool
    IsAsync           bool

    // NEW: For ProviderTypeStruct
    StructType   types.Type         // The struct type T (e.g., *Config)
    StructFields []*StructFieldSpec // Extracted fields, sorted alphabetically

    // NEW: For ProviderTypeFieldAccess (synthetic providers created in graph)
    SourceField *StructFieldSpec // The field this accessor extracts
}
```

### 2.4 InjectorFieldAccessStmt

**Location**: `internal/kessoku/provider.go` (NEW)

```go
// InjectorFieldAccessStmt represents field extraction from a struct instance.
// Implements InjectorStmt interface.
type InjectorFieldAccessStmt struct {
    StructParam *InjectorParam   // The struct instance parameter
    Field       *StructFieldSpec // The field to extract
    ReturnParam *InjectorParam   // The result parameter
}

// Stmt generates: fieldVar := structVar.FieldName
func (stmt *InjectorFieldAccessStmt) Stmt(varPool *VarPool, injector *Injector, returnErrStmts func(errExpr ast.Expr) []ast.Stmt) ([]ast.Stmt, []string) {
    // Implementation in research.md section 2.3
}

// HasAsync returns false - field access is always synchronous
func (stmt *InjectorFieldAccessStmt) HasAsync() bool {
    return false
}
```

---

## 3. Integration Points

### 3.1 Parser Integration

**File**: `internal/kessoku/parser.go`

**Integration Point**: `parseProviderType()` switch statement (line ~446)

**Current Code**:
```go
switch named.Obj().Name() {
case "bindProvider":
    // ...
case "asyncProvider":
    // ...
case "fnProvider":
    // ...
}
```

**New Case**:
```go
case "structProvider":
    if typeArgs.Len() < 1 {
        return nil, nil, false, false, fmt.Errorf("structProvider requires 1 type argument")
    }

    structType := typeArgs.At(0)

    // Validate struct type
    underlying := structType
    if ptr, ok := structType.Underlying().(*types.Pointer); ok {
        underlying = ptr.Elem()
    }

    structUnderlying, ok := underlying.Underlying().(*types.Struct)
    if !ok {
        return nil, nil, false, false, fmt.Errorf("not a struct type: %s", structType)
    }

    // Field extraction handled separately - metadata stored in ProviderSpec
    // Returns: requires struct type, provides nothing (expanded in graph)
    return []types.Type{structType}, nil, false, false, nil
```

**Integration Point**: `parseProviderArgument()` (line ~347)

After parsing, populate `ProviderSpec.StructType` and `ProviderSpec.StructFields`:
```go
// After parseProviderType call
if named.Obj().Name() == "structProvider" {
    providerSpec.Type = ProviderTypeStruct
    providerSpec.StructType = structType
    providerSpec.StructFields = extractExportedFields(structUnderlying)
}
```

### 3.2 Graph Integration

**File**: `internal/kessoku/graph.go`

**Integration Point**: `NewGraph()` function (line ~261)

**Current Code**:
```go
fnProviderMap := make(map[string]*fnProvider)
for _, provider := range build.Providers {
    for groupIndex, typeGroup := range provider.Provides {
        // ... register providers
    }
}
```

**Modified Code**:
```go
fnProviderMap := make(map[string]*fnProvider)

// First pass: register function providers
for _, provider := range build.Providers {
    if provider.Type == ProviderTypeStruct {
        continue // Handle in second pass
    }
    for groupIndex, typeGroup := range provider.Provides {
        // ... existing logic
    }
}

// Second pass: expand struct providers (maintains declaration order)
for _, provider := range build.Providers {
    if provider.Type != ProviderTypeStruct {
        continue
    }

    // Validate struct provider exists (FR-010)
    structTypeKey := provider.StructType.String()
    structProvider, hasStructProvider := fnProviderMap[structTypeKey]
    if !hasStructProvider {
        // Check for type mismatch (FR-006)
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

    // Create synthetic field accessor providers (FR-012: fields already sorted)
    for _, field := range provider.StructFields {
        fieldTypeKey := field.Type.String()

        // Check for duplicates (FR-005)
        if _, exists := fnProviderMap[fieldTypeKey]; exists {
            return nil, fmt.Errorf("multiple providers provide %s", fieldTypeKey)
        }

        // Create synthetic provider
        syntheticProvider := &ProviderSpec{
            Type:          ProviderTypeFieldAccess,
            Requires:      []types.Type{provider.StructType},
            Provides:      [][]types.Type{{field.Type}},
            SourceField:   field,
            IsReturnError: false,
            IsAsync:       false,
        }

        fnProviderMap[fieldTypeKey] = &fnProvider{
            provider:    syntheticProvider,
            returnIndex: 0,
        }
    }
}
```

### 3.3 Generator Integration

**File**: `internal/kessoku/generator.go`

**Integration Point**: Statement generation loop (line ~435)

**Current Code**:
```go
for _, stmt := range injector.Stmts {
    newStmts, _ := stmt.Stmt(varPool, injector, returnErrStmts)
    stmts = append(stmts, newStmts...)
}
```

**No change needed** - the `InjectorStmt` interface handles polymorphism.

**Integration Point**: New statement type handling

The `InjectorFieldAccessStmt.Stmt()` method generates:
```go
// For field DBHost of type string from *Config
dbHost := config.DBHost

// Generated AST:
&ast.AssignStmt{
    Lhs: []ast.Expr{ast.NewIdent("dbHost")},
    Tok: token.DEFINE,  // or ASSIGN in async context
    Rhs: []ast.Expr{
        &ast.SelectorExpr{
            X:   ast.NewIdent("config"),
            Sel: ast.NewIdent("DBHost"),
        },
    },
}
```

### 3.4 Import Collection Integration

**Integration Point**: `collectImportsFromType()` (line ~126 in `provider.go`)

**Already handles** struct fields via `*types.Struct` case:
```go
case *types.Struct:
    for i := 0; i < typ.NumFields(); i++ {
        collectImportsFromType(typ.Field(i).Type(), pkg, imports, referencedImports, varPool)
    }
```

For field accessor parameters, use `NewInjectorParamWithImports()`:
```go
fieldParam := NewInjectorParamWithImports(
    []types.Type{field.Type},
    false, // not an arg
    metaData.Package.Path,
    metaData.Imports,
    varPool,
)
```

---

## 4. Entity Relationships

```
┌──────────────────────────────────────────────────────────────────┐
│                        BuildDirective                             │
│  (represents a kessoku.Inject call)                              │
├──────────────────────────────────────────────────────────────────┤
│  InjectorName: string                                             │
│  Return: *Return                                                  │
│  Providers: []*ProviderSpec ──────────────────────────┐          │
└──────────────────────────────────────────────────────────────────┘
                                                         │
                        ┌────────────────────────────────┴──────────────────────────────┐
                        │                                                                │
                        ▼                                                                ▼
┌───────────────────────────────────────┐     ┌───────────────────────────────────────────────┐
│         ProviderSpec (function)        │     │         ProviderSpec (struct)                  │
│  Type: ProviderTypeFunction            │     │  Type: ProviderTypeStruct                      │
│  Provides: [[*Config]]                 │     │  StructType: *Config                           │
│  Requires: []                          │     │  StructFields: []*StructFieldSpec ─────────┐  │
│  ASTExpr: kessoku.Provide(NewConfig)   │     │  Provides: nil (expanded in graph)         │  │
└───────────────────────────────────────┘     └──────────────────────────────────────────────┘
                                                                                            │
                        ┌───────────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                       StructFieldSpec                             │
│  (represents an extracted struct field)                          │
├──────────────────────────────────────────────────────────────────┤
│  Name: "DBHost"          // Used in generated code               │
│  Type: string            // Used for dependency matching         │
│  Index: 0                // Original position (preserved)        │
│  Anonymous: false        // True for embedded fields             │
└──────────────────────────────────────────────────────────────────┘
                        │
                        │ (expanded in graph.go)
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│         ProviderSpec (field_access) - SYNTHETIC                   │
│  Type: ProviderTypeFieldAccess                                   │
│  Requires: [*Config]      // Depends on struct                   │
│  Provides: [[string]]     // Provides field type                 │
│  SourceField: *StructFieldSpec                                   │
└──────────────────────────────────────────────────────────────────┘
                        │
                        │ (converted in Build())
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                  InjectorFieldAccessStmt                          │
│  (implements InjectorStmt)                                       │
├──────────────────────────────────────────────────────────────────┤
│  StructParam: *InjectorParam  // "config"                        │
│  Field: *StructFieldSpec      // DBHost info                     │
│  ReturnParam: *InjectorParam  // "dbHost"                        │
└──────────────────────────────────────────────────────────────────┘
                        │
                        │ Stmt() generates
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Generated Code                                │
│  dbHost := config.DBHost                                         │
└──────────────────────────────────────────────────────────────────┘
```

---

## 5. State Transitions

### 5.1 Parse Stage

```
Source Code                          ProviderSpec
─────────────────                    ────────────
kessoku.Struct[*Config]()    →       Type: ProviderTypeStruct
                                     StructType: *Config
                                     StructFields: [{Name:"DBHost", Type:string, Index:0},
                                                    {Name:"DBPort", Type:int, Index:1}]
                                     Requires: [*Config]
                                     Provides: nil
```

### 5.2 Graph Stage

```
ProviderSpec (struct)                Synthetic ProviderSpec (field_access) × N
─────────────────────                ──────────────────────────────────────────
StructFields: [DBHost, DBPort]   →   1. Type: ProviderTypeFieldAccess
                                        Requires: [*Config]
                                        Provides: [[string]]
                                        SourceField: DBHost

                                     2. Type: ProviderTypeFieldAccess
                                        Requires: [*Config]
                                        Provides: [[int]]
                                        SourceField: DBPort
```

### 5.3 Build Stage

```
Synthetic ProviderSpec               InjectorFieldAccessStmt
──────────────────────               ───────────────────────
SourceField: DBHost              →   StructParam: config (from graph)
                                     Field: DBHost
                                     ReturnParam: dbHost (new)
```

### 5.4 Generate Stage

```
InjectorFieldAccessStmt              Generated Code
───────────────────────              ──────────────
StructParam: config              →   dbHost := config.DBHost
Field: DBHost
ReturnParam: dbHost
```

---

## 6. Validation Rules

### 6.1 Parse-Time Validation

| Rule | Location | Error Message |
|------|----------|---------------|
| Type argument provided | `parseProviderType` | "structProvider requires 1 type argument" |
| T is struct or *struct | `parseProviderType` | "not a struct type: `<type>`" |

### 6.2 Graph-Time Validation

| Rule | Location | Error Message |
|------|----------|---------------|
| Struct provider exists | `NewGraph` | "no provider for type `<type>`" |
| Exact type match | `NewGraph` | "type mismatch: expected `<expected>`, got `<actual>`" |
| No duplicate field types | `NewGraph` | "multiple providers provide `<type>`" |

### 6.3 Field-Level Rules

| Rule | Behavior |
|------|----------|
| Unexported fields | Silently filtered during extraction |
| Unexported embedded types | Silently filtered (field.Exported() check) |
| No exported fields | Empty StructFields list (silent success) |
| Field ordering | Alphabetical by Name, preserved Index |

---

## 7. Memory and Lifecycle

### 7.1 Synthetic Provider Lifecycle

- Created during `NewGraph()` construction
- Stored in `fnProviderMap` for dependency resolution
- Converted to `InjectorFieldAccessStmt` during `Build()`
- Garbage collected after code generation

### 7.2 StructFieldSpec Sharing

- Field specs are created once during parsing
- Referenced by both struct provider and synthetic field accessor
- No deep copy needed (immutable data)

---

## 8. Import Handling

When a field type comes from an external package:

1. The import is collected during struct field extraction via `collectImportsFromType()`
2. Added to `ProviderSpec.ReferencedImports`
3. For synthetic field accessors, imports are collected via `NewInjectorParamWithImports()`
4. Marked as used during code generation
5. Included in generated file imports

**Example**:
```go
// Source struct
type Config struct {
    Logger *log.Logger  // External package
    DBHost string       // Built-in type
}

// Generated imports
import (
    "log"
)
```
