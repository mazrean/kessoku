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

# Lint (mandatory before commit)
go tool tools lint ./...

# Code generation
go generate ./...                          # Generate DI code via go:generate
go tool kessoku [files...]                 # Direct codegen for specific files

# API compatibility check
go tool tools apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku

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

### Key Code Locations

- `annotation.go`: Public API (`Inject`, `Provide`, `Async`, `Bind`, `Value`, `Set`, `Struct`)
- `internal/kessoku/provider.go`: Core data structures (`ProviderSpec`, `Injector`, `InjectorStmt`)
- `internal/config/`: CLI configuration and orchestration

## Development Guidelines

### Mandatory Before Commit
- Run `go tool tools lint ./...` - treat failures as blockers
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
