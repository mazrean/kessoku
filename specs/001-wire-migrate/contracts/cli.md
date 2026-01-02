# CLI Contract: kessoku migrate

**Version**: 1.0.0
**Date**: 2026-01-02

## Command Synopsis

```
kessoku migrate [flags] [patterns...]
```

## Description

Migrate google/wire configuration files to kessoku format. The tool parses wire patterns and generates equivalent kessoku code. It uses the `wireinject` build tag to load wire configuration files (same as wire itself).

## Arguments

| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `patterns` | No | `./` | Go package patterns to migrate (e.g., `./`, `./pkg/...`, `example.com/pkg`)

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `kessoku.go` | Output file path |
| `--log-level` | `-l` | `info` | Log level: debug, info, warn, error |
| `--help` | `-h` | - | Show help message |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (syntax error, type resolution error, package mismatch, name collision, missing constructor) |

## Output Behavior

### Default (no `-o` flag)
Output written to `kessoku.go` in current directory.

### With `-o` flag
Output written to specified file path.

### No Output Conditions
Output file is NOT generated when:
- No wire import found in any input file (warning emitted, file skipped)
- No wire patterns found in any input file after processing (warning emitted)

### Multiple Input Files
All conversions merged into single output file with:
- Deduplicated imports
- All variable declarations preserved
- Package declaration from first file
- Build tags and comments from wire files are NOT preserved (wire-specific)

## Stdout Output

- Success: No output (silent)
- With `--log-level=info`: Progress messages to stderr
- With `--log-level=debug`: Detailed parsing/transformation info to stderr

## Stderr Output

### Info Level Messages

```
INFO Migrating wire configuration patterns=[./]
INFO Generated kessoku configuration output=kessoku.go
```

### Warning Messages

```
WARN No wire import found file=notwire.go
WARN No wire patterns found file=empty.go
WARN Unsupported pattern pattern=wire.Build location=wire.go:15:2
```

### Error Messages

```
ERROR Syntax error file=broken.go message="/path/to/broken.go:10:5: unexpected token"
ERROR Type resolution failed file=wire.go message="/path/to/wire.go:15:2: undefined: SomeType"
ERROR Missing constructor file=wire.go message="/path/to/wire.go:20:5: no constructor NewPostgresRepo found for type PostgresRepo"
ERROR Package mismatch files=[a.go b.go] packages=[pkg1 pkg2]
ERROR Name collision identifier=FooSet files=[a.go b.go]
```

## Examples

### Basic Migration

```bash
# Migrate current directory (default)
kessoku migrate

# Migrate specific package
kessoku migrate ./pkg/di

# Migrate all packages recursively
kessoku migrate ./...

# Migrate with custom output
kessoku migrate -o providers.go ./pkg/di

# Verbose output
kessoku migrate -l debug ./...
```

### Expected Transformations

Input (`wire.go`):
```go
package app

import "github.com/google/wire"

var SuperSet = wire.NewSet(
    NewDatabase,
    wire.Bind(new(Repository), new(*PostgresRepo)),
    wire.Value("config-value"),
)
```

Output (`kessoku.go`):
```go
package app

import "github.com/mazrean/kessoku"

var SuperSet = kessoku.Set(
    kessoku.Provide(NewDatabase),
    kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
    kessoku.Value("config-value"),
)
```

## Error Conditions

### Package Mismatch (Exit 1)
When input files belong to different packages:
```
ERROR Package mismatch files=[a.go b.go] packages=[pkg1 pkg2]
```

### Name Collision (Exit 1)
When same identifier appears in multiple files:
```
ERROR Name collision identifier=FooSet files=[a.go b.go]
```

### Syntax Error (Exit 1)
When input file has Go syntax errors:
```
ERROR Syntax error file=broken.go message="/path/to/broken.go:10:5: unexpected token"
```

### Type Resolution Failure (Exit 1)
When types cannot be resolved (missing imports, undefined types):
```
ERROR Type resolution failed file=wire.go message="/path/to/wire.go:15:2: undefined: SomeType"
```

### Missing Constructor (Exit 1)
When wire.Bind references a type without a discoverable constructor:
```
ERROR Missing constructor file=wire.go message="/path/to/wire.go:20:5: no constructor NewPostgresRepo found for type PostgresRepo"
```
