# Quickstart: Wire to Kessoku Migration Tool

**Date**: 2026-01-02
**Estimated Implementation Time**: Not specified (see tasks.md for breakdown)

## Prerequisites

- Go 1.24+
- Familiarity with go/ast and go/types packages
- Understanding of kessoku's existing codebase structure

## Implementation Overview

This feature adds a `migrate` subcommand to the kessoku CLI that converts google/wire configuration files to kessoku format.

## Quick Implementation Path

### Step 1: Add CLI Subcommand

Modify `internal/config/config.go` to use Kong's subcommand pattern:

```go
type CLI struct {
    LogLevel string          `short:"l" default:"info"`
    Generate GenerateCmd     `cmd:"" default:"1"`
    Migrate  MigrateCmd      `cmd:""`
    Version  kong.VersionFlag
}

type MigrateCmd struct {
    Output string   `short:"o" default:"kessoku.go"`
    Files  []string `arg:""`
}
```

### Step 2: Create Migration Package

Create `internal/migrate/` with these files:

```
internal/migrate/
├── migrate.go       # Orchestrator
├── parser.go        # Wire pattern detection
├── transformer.go   # Wire → Kessoku conversion
├── writer.go        # Output generation
└── patterns.go      # Pattern type definitions
```

### Step 3: Implement Core Flow

```go
// migrate.go

// MigrateFiles orchestrates the migration of wire files to kessoku format
func (m *Migrator) MigrateFiles(files []string) error {
    // 1. Load packages with type info
    pkgs, err := packages.Load(cfg, files...)
    if err != nil {
        return fmt.Errorf("failed to load packages: %w", err)
    }

    // Check for load errors - any error is fatal
    for _, pkg := range pkgs {
        if len(pkg.Errors) > 0 {
            return m.convertPackageError(pkg.Errors[0])
        }
    }

    // 2. Extract wire patterns from each file
    var results []MigrationResult
    for _, pkg := range pkgs {
        for _, file := range pkg.Syntax {
            patterns := m.parser.ExtractPatterns(file, pkg.TypesInfo)

            // Transform patterns - may return error (e.g., missing constructor)
            kessokuPatterns, err := m.transformer.Transform(patterns)
            if err != nil {
                return err // Propagate transformation errors
            }

            results = append(results, MigrationResult{...})
        }
    }

    // 3. Merge results - may return error (package mismatch, name collision)
    merged, err := m.mergeResults(results)
    if err != nil {
        return err // Propagate merge errors
    }

    // 4. Write output
    return m.writer.Write(merged, outputPath)
}

// convertPackageError converts packages.Error to ParseError
func (m *Migrator) convertPackageError(pkgErr packages.Error) error {
    // Classify by packages.Error.Kind
    kind := ParseErrorSyntax
    if pkgErr.Kind == packages.TypeError {
        kind = ParseErrorTypeResolution
    }

    // Preserve the full position string in the message for CLI output
    // Format: "/path/file.go:line:col: error message"
    message := pkgErr.Msg
    if pkgErr.Pos != "" {
        message = pkgErr.Pos + ": " + pkgErr.Msg
    }

    // Extract file path from position string
    // Format: "/path/file.go:line:col" or "C:\path\file.go:line:col" (Windows)
    // Find the .go extension and include everything up to and including it
    file := ""
    if pkgErr.Pos != "" {
        if idx := strings.Index(pkgErr.Pos, ".go:"); idx > 0 {
            file = pkgErr.Pos[:idx+3] // Include ".go"
        }
    }

    return &ParseError{
        Kind:    kind,
        File:    file,
        Pos:     token.NoPos, // Full location preserved in Message
        Message: message,
    }
}
```

### Step 4: Implement Pattern Detection

Use AST visitor to find wire patterns:

```go
func (p *Parser) ExtractPatterns(file *ast.File, info *types.Info) []WirePattern {
    var patterns []WirePattern

    ast.Inspect(file, func(n ast.Node) bool {
        call, ok := n.(*ast.CallExpr)
        if !ok {
            return true
        }

        if isWireNewSet(call) {
            patterns = append(patterns, p.parseNewSet(call, info))
        } else if isWireBind(call) {
            patterns = append(patterns, p.parseBind(call, info))
        }
        // ... other patterns

        return true
    })

    return patterns
}
```

### Step 5: Implement Transformations

Transform each wire pattern to kessoku equivalent:

```go
func (t *Transformer) Transform(patterns []WirePattern) ([]KessokuPattern, error) {
    var result []KessokuPattern

    for _, p := range patterns {
        switch p := p.(type) {
        case *WireNewSet:
            transformed, err := t.transformNewSet(p)
            if err != nil {
                return nil, err
            }
            result = append(result, transformed)
        case *WireBind:
            transformed, err := t.transformBind(p)
            if err != nil {
                return nil, err // e.g., missing constructor
            }
            result = append(result, transformed)
        case *WireStruct:
            result = append(result, t.transformStruct(p))
        // ...
        }
    }

    return result, nil
}
```

## Key Patterns

### wire.Bind → kessoku.Bind

```go
// Input: wire.Bind(new(Repository), new(*PostgresRepo))
// Need to find constructor for PostgresRepo

func (t *Transformer) transformBind(b *WireBind) (*KessokuBind, error) {
    // Handle pointer types: new(*PostgresRepo) gives **PostgresRepo
    // We need to unwrap to get the base named type
    implType := b.Implementation
    for {
        if ptr, ok := implType.(*types.Pointer); ok {
            implType = ptr.Elem()
        } else {
            break
        }
    }

    named, ok := implType.(*types.Named)
    if !ok {
        // Handle error: implementation must be a named type
        return nil, fmt.Errorf("implementation type must be a named type, got %T", implType)
    }

    typeName := named.Obj().Name()

    // Look up constructor in package scope
    // Convention: NewTypeName
    pkg := named.Obj().Pkg()
    constructorName := "New" + typeName
    obj := pkg.Scope().Lookup(constructorName)
    if obj == nil {
        // No constructor found - this is an error
        // wire.Bind requires an existing provider or constructor
        return nil, fmt.Errorf("no constructor %q found for type %q in package %q",
            constructorName, typeName, pkg.Path())
    }

    // Verify it's a function
    if _, ok := obj.(*types.Func); !ok {
        return nil, fmt.Errorf("%q is not a function", constructorName)
    }

    return &KessokuBind{
        Interface: b.Interface,
        Provider: &KessokuProvide{
            FuncExpr: ast.NewIdent(constructorName),
        },
    }, nil
}
```

### wire.Struct → kessoku.Provide + func literal

```go
// Input: wire.Struct(new(Config), "*")
// Output: kessoku.Provide(func(f1 T1, f2 T2) *Config { return &Config{F1: f1, F2: f2} })

func (t *Transformer) transformStruct(s *WireStruct) *KessokuProvide {
    structType := s.StructType.Underlying().(*types.Struct)

    // Build function parameters and struct literal
    // NOTE: Include ALL fields (exported and unexported) since migration
    // generates code in the same package where unexported fields are accessible
    var params, fields []string
    for i := 0; i < structType.NumFields(); i++ {
        field := structType.Field(i)
        // For "*", include all fields; otherwise check if field is in the list
        if s.Fields[0] != "*" && !contains(s.Fields, field.Name()) {
            continue
        }
        paramName := strings.ToLower(field.Name()[:1]) + field.Name()[1:]
        params = append(params, paramName)
        fields = append(fields, field.Name())
    }

    // Generate func literal AST
    // ...
}
```

## Testing Strategy

Create golden file tests:

```
testdata/
├── basic/
│   ├── input.go
│   └── expected.go
├── bind/
│   ├── input.go
│   └── expected.go
└── struct/
    ├── input.go
    └── expected.go
```

Run with:

```go
func TestMigration(t *testing.T) {
    entries, _ := os.ReadDir("testdata")
    for _, entry := range entries {
        t.Run(entry.Name(), func(t *testing.T) {
            input := filepath.Join("testdata", entry.Name(), "input.go")
            expected := filepath.Join("testdata", entry.Name(), "expected.go")

            result := migrate(input)
            expectedContent, _ := os.ReadFile(expected)

            if result != string(expectedContent) {
                t.Errorf("mismatch for %s", entry.Name())
            }
        })
    }
}
```

## Common Pitfalls

1. **Import aliasing**: Wire files may use `import w "github.com/google/wire"` - handle selector expressions like `w.NewSet`

2. **Type resolution**: Need `packages.NeedTypes | packages.NeedTypesInfo` to resolve types for Bind/Struct transformations

3. **Pointer types**: `new(T)` in wire gives `*T` - track `IsPointer` for proper code generation

4. **Nested sets**: `wire.NewSet(OtherSet, ...)` may reference other variables - preserve as-is

## Next Steps

After basic implementation:
1. Run `go tool tools lint ./...` to check for issues
2. Run `go test -v ./internal/migrate/...`
3. Test with real wire files from examples
