# Data Model: Wire to Kessoku Migration Tool

**Date**: 2026-01-02
**Status**: Complete

## Core Entities

### 1. WirePattern (Abstract)

Base representation for all detected wire patterns in source code.

| Field | Type | Description |
|-------|------|-------------|
| `Kind` | `WirePatternKind` | Pattern type discriminator |
| `Pos` | `token.Pos` | Source position for error reporting |
| `File` | `string` | Source file path |

**Validation Rules**:
- `Pos` must be valid (> 0)
- `File` must exist and be readable

### 2. WireNewSet

Represents `wire.NewSet(...)` pattern.

| Field | Type | Description |
|-------|------|-------------|
| `VarName` | `string` | Variable name (e.g., "DatabaseSet") |
| `Elements` | `[]WirePattern` | Nested patterns within the set |

**Validation Rules**:
- `VarName` must be a valid Go identifier
- `Elements` may be empty (warning case)

### 3. WireBind

Represents `wire.Bind(new(Interface), new(Impl))` pattern.

| Field | Type | Description |
|-------|------|-------------|
| `Interface` | `types.Type` | Interface type being bound |
| `Implementation` | `types.Type` | Concrete type implementing interface |

**Validation Rules**:
- `Interface` must be an interface type
- `Implementation` must implement `Interface`

### 4. WireValue

Represents `wire.Value(expr)` pattern.

| Field | Type | Description |
|-------|------|-------------|
| `Expr` | `ast.Expr` | Value expression |
| `Type` | `types.Type` | Resolved type of the value |

**Validation Rules**:
- `Type` must not be an interface type (wire constraint)

### 5. WireInterfaceValue

Represents `wire.InterfaceValue(new(Interface), expr)` pattern.

| Field | Type | Description |
|-------|------|-------------|
| `Interface` | `types.Type` | Interface type to provide |
| `Expr` | `ast.Expr` | Value expression |

**Validation Rules**:
- `Interface` must be an interface type

### 6. WireStruct

Represents `wire.Struct(new(Type), fields...)` pattern.

| Field | Type | Description |
|-------|------|-------------|
| `StructType` | `types.Type` | Struct type to construct |
| `Fields` | `[]string` | Field names, or ["*"] for all |
| `IsPointer` | `bool` | Whether new(T) was used |

**Validation Rules**:
- `StructType` must be a struct type
- `Fields` entries must exist in struct (if not "*")

### 7. WireFieldsOf

Represents `wire.FieldsOf(new(Type), fields...)` pattern.

| Field | Type | Description |
|-------|------|-------------|
| `StructType` | `types.Type` | Source struct type |
| `Fields` | `[]string` | Field names to extract |

**Validation Rules**:
- `StructType` must be a struct type
- `Fields` must not be empty
- All field names must exist in struct

### 8. WireProviderFunc

Represents a provider function reference within a set.

| Field | Type | Description |
|-------|------|-------------|
| `Func` | `*types.Func` | Function object |
| `Name` | `string` | Function name as written |

**Validation Rules**:
- `Func` must have at least one return value

---

## Output Entities

### 9. KessokuPattern (Abstract)

Base for generated kessoku patterns.

| Field | Type | Description |
|-------|------|-------------|
| `Kind` | `KessokuPatternKind` | Pattern type |
| `SourcePos` | `token.Pos` | Original wire source position |

### 10. KessokuSet

Generated `kessoku.Set(...)`.

| Field | Type | Description |
|-------|------|-------------|
| `VarName` | `string` | Variable name |
| `Elements` | `[]KessokuPattern` | Providers in the set |

### 11. KessokuProvide

Generated `kessoku.Provide(fn)`.

| Field | Type | Description |
|-------|------|-------------|
| `FuncExpr` | `ast.Expr` | Function expression or literal |

### 12. KessokuBind

Generated `kessoku.Bind[I](provider)`.

| Field | Type | Description |
|-------|------|-------------|
| `Interface` | `types.Type` | Interface type parameter |
| `Provider` | `KessokuPattern` | Wrapped provider |

### 13. KessokuValue

Generated `kessoku.Value(expr)`.

| Field | Type | Description |
|-------|------|-------------|
| `Expr` | `ast.Expr` | Value expression |

---

## Orchestration Entities

### 14. MigrationResult

Result of migrating a single file.

| Field | Type | Description |
|-------|------|-------------|
| `SourceFile` | `string` | Original file path |
| `Package` | `string` | Package name |
| `Imports` | `[]ImportSpec` | Required imports |
| `Patterns` | `[]KessokuPattern` | Generated patterns |
| `Warnings` | `[]Warning` | Non-fatal issues |

### 15. MergedOutput

Result of merging multiple file migrations.

| Field | Type | Description |
|-------|------|-------------|
| `Package` | `string` | Package name |
| `Imports` | `[]ImportSpec` | Deduplicated imports |
| `TopLevelDecls` | `[]ast.Decl` | Variable declarations |

**Validation Rules**:
- All source files must have same package name
- No duplicate identifier names across files

### 16. Warning

Non-fatal issue during migration.

| Field | Type | Description |
|-------|------|-------------|
| `Pos` | `token.Pos` | Source location |
| `Message` | `string` | Warning description |
| `Code` | `WarningCode` | Warning type enum |

### 17. ImportSpec

Import declaration for output file.

| Field | Type | Description |
|-------|------|-------------|
| `Path` | `string` | Import path (e.g., "github.com/mazrean/kessoku") |
| `Name` | `string` | Optional alias (empty for default) |

**Validation Rules**:
- `Path` must be a valid Go import path

### 18. ParseError

Error encountered during single-file parsing/analysis.

| Field | Type | Description |
|-------|------|-------------|
| `Kind` | `ParseErrorKind` | Error type |
| `File` | `string` | File where error occurred |
| `Pos` | `token.Pos` | Position in file (may be NoPos if unavailable) |
| `Message` | `string` | Human-readable description |

**Validation Rules**:
- `File` should be a valid file path when available; may be empty for load-time errors
- `Pos` may be token.NoPos when exact position unavailable
- `Message` must contain sufficient context for CLI output (includes position if Pos is NoPos)

### 19. MergeError

Error encountered during multi-file merging.

| Field | Type | Description |
|-------|------|-------------|
| `Kind` | `MergeErrorKind` | Error type |
| `Message` | `string` | Human-readable description |
| `Files` | `[]string` | Files involved in the error |
| `Identifier` | `string` | Conflicting identifier (for name collision) |
| `Packages` | `[]string` | Package names (for package mismatch) |

**Validation Rules**:
- `Files` must have at least 2 entries
- `Identifier` required when `Kind` is `MergeErrorNameCollision`
- `Packages` required when `Kind` is `MergeErrorPackageMismatch`

---

## Enums

### WirePatternKind

```go
const (
    PatternNewSet WirePatternKind = iota
    PatternBind
    PatternValue
    PatternInterfaceValue
    PatternStruct
    PatternFieldsOf
    PatternProviderFunc
    PatternUnsupported
)
```

### WarningCode

```go
const (
    WarnNoWireImport WarningCode = iota
    WarnNoWirePatterns
    WarnUnsupportedPattern
)
```

### ParseErrorKind

```go
const (
    ParseErrorSyntax ParseErrorKind = iota
    ParseErrorTypeResolution
    ParseErrorMissingConstructor
)
```

### MergeErrorKind

```go
const (
    MergeErrorPackageMismatch MergeErrorKind = iota
    MergeErrorNameCollision
)
```

---

## State Transitions

### File Processing State

```
[Input File]
    → Parse
    → [ast.File + types.Package]
    → Extract Patterns
    → [[]WirePattern]
    → Transform
    → [MigrationResult]
    → (repeat for each file)
    → Merge
    → [MergedOutput]
    → Generate
    → [Output File]
```

### Pattern Transformation Flow

```
WireNewSet      → KessokuSet
WireBind        → KessokuBind + KessokuProvide
WireValue       → KessokuValue
WireInterfaceValue → KessokuBind + KessokuValue
WireStruct      → KessokuProvide (with func literal)
WireFieldsOf    → []KessokuProvide (one per field)
WireProviderFunc → KessokuProvide
```
