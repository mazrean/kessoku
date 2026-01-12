# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kessoku is a compile-time dependency injection library for Go that speeds up application startup through parallel dependency injection. Unlike google/wire which initializes services sequentially, Kessoku automatically executes independent providers in parallel. It generates optimized code at compile time with zero runtime overhead.

## Common Commands

```bash
# Build
go build -o bin/kessoku ./cmd/kessoku

# Test
go test -v ./...                           # Run all tests
go test -v -run TestName ./...             # Run specific test
go test -v ./internal/kessoku/...          # Run tests in specific package

# Golden tests (code generation validation)
go test -v -run TestGoldenGeneration ./internal/kessoku/...           # Run golden tests
go test -v -run TestGoldenGeneration ./internal/kessoku/... -update   # Update golden files

# Lint (mandatory before commit)
go tool lint ./...
go tool lint -fix ./...                # Auto-fix issues where possible

# Code generation
go generate ./...                          # Generate DI code via go:generate
go tool kessoku [files...]                 # Direct codegen for specific files

# Wire migration
go tool kessoku migrate [patterns...] -o kessoku.go    # Migrate wire config to kessoku (default: ./)

# API compatibility check
go tool apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku

# Release
go tool goreleaser release --snapshot --clean  # Snapshot (local testing)
```

## Architecture

### Module Structure

Go 1.24 workspace with two modules:
- **Main module** (`github.com/mazrean/kessoku`): Public API and codegen engine
- **Tools module** (`./tools`): Custom linter combining govet + staticcheck + stylecheck analyzers

### Code Generation Pipeline

The codegen engine in `internal/kessoku/` follows this flow:

```
parser.go → graph.go → generator.go
   ↓            ↓            ↓
Parse AST   Build DAG    Emit code
& extract   & detect     with parallel
providers   cycles       execution
```

1. **Parser**: Finds `kessoku.Inject` calls, extracts provider types and dependencies
2. **Graph**: Constructs dependency DAG, detects cycles, computes parallel execution pools
3. **Generator**: Emits `*_band.go` files with optimized injector functions

### Wire Migration Tool

The `kessoku migrate` command converts google/wire configuration files to kessoku format.
It uses the `wireinject` build tag to load wire configuration files (same as wire itself).

```bash
# Basic usage (migrate current directory)
go tool kessoku migrate

# Migrate specific package
go tool kessoku migrate ./pkg/di

# Migrate with custom output
go tool kessoku migrate ./... -o providers.go
```

Supported wire patterns:
- `wire.NewSet(providers...)` → `kessoku.Set(providers...)`
- `wire.Bind(new(Interface), new(Impl))` → `kessoku.Bind[Interface]()`
- `wire.Value(v)` → `kessoku.Value(v)`
- `wire.InterfaceValue(new(I), v)` → `kessoku.Bind[I](kessoku.Value(v))`
- `wire.Struct(new(T), "Field1", "Field2")` → `kessoku.Provide(func(f1, f2) *T { ... })`
- `wire.FieldsOf(new(T), "F1", "F2")` → `kessoku.Provide(func(t *T) (T1, T2) { ... })`
- Set references (e.g., `wire.NewSet(OtherSet, ...)`) are preserved

Migration tool location: `internal/migrate/`

### Key Code Locations

- `annotation.go`: Public API (`Inject`, `Provide`, `Async`, `Bind`, `Value`, `Set`, `Struct`)
- `internal/kessoku/provider.go`: Core data structures (`ProviderSpec`, `Injector`, `InjectorStmt`)
- `internal/kessoku/golden_test.go`: Golden tests for code generation validation
- `internal/kessoku/testdata/`: Test cases for golden tests (input files + expected.go)
- `internal/config/`: CLI configuration and orchestration
- `internal/migrate/`: Wire to Kessoku migration tool

## Development Guidelines

### Mandatory Before Commit
- Run `go tool lint ./...` - treat failures as blockers
- Run `go test -v ./...` - all tests must pass
- Run `go fmt ./...` - format code

### Development Methodology
- Follow TDD approach
- One logical change per commit
- Commit style: `fix:`, `feat:`, `chore:`, `deps:`, `ci:`

### Documentation Rules
- Update CLAUDE.md when architecture or commands change
- Update README.md when user-facing features change
- Documentation updates should be part of the same commit as related code changes

## Active Technologies
- Go 1.24+ + github.com/alecthomas/kong (CLI), golang.org/x/tools (AST parsing, type checking) (001-wire-migrate)
- N/A (file-based input/output, no persistent storage) (001-wire-migrate)
- Go 1.24+ + github.com/alecthomas/kong (CLI framework, already in use) (002-agent-skills-setup)
- File-based output (Skills files installed to filesystem) (002-agent-skills-setup)
- Go 1.24+ + golang.org/x/tools (AST/type checking), standard library (testing, flag, os, path/filepath) (003-golden-test)
- File-based (testdata directory with input/expected files) (003-golden-test)
- Go 1.24+ + github.com/alecthomas/kong (CLI framework) (001-expand-agent-support)
- N/A (file-based skill installation) (001-expand-agent-support)

## Recent Changes
- 001-wire-migrate: Added Go 1.24+ + github.com/alecthomas/kong (CLI), golang.org/x/tools (AST parsing, type checking)
