# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kessoku is a dependency injection CLI tool and library for Go, similar to google/wire but with advanced async provider support. It generates Go code for dependency injection based on provider functions and injector declarations. The tool performs compile-time dependency injection through code generation, eliminating runtime reflection overhead.

### Key Differentiators from google/wire

- **Async Provider Support**: Parallel execution of independent providers using `kessoku.Async()`
- **Context Injection**: Automatic `context.Context` injection for async operations
- **Channel Synchronization**: Advanced coordination between dependent async providers
- **Error Handling**: Comprehensive error propagation across async boundaries
- **Performance**: Optimized code generation for minimal memory allocations

## Common Commands

### Building
```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Generate dependency injection code using go generate
go generate ./...

# Generate dependency injection code directly
go tool kessoku [files...]
```

### Testing
```bash
# Run tests
go test -v ./...

# Format code
go fmt ./...
```

### Linting
```bash
# Run comprehensive Go analyzer linter
go tool tools lint ./...
```

### API Compatibility
```bash
# Check API compatibility against a previous version
go tool tools apicompat <base_package_path> <target_package_path>

# Example: Check current changes against latest released version
go tool tools apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku

# Example: Check against a specific version
go tool tools apicompat github.com/mazrean/kessoku@v1.0.0 github.com/mazrean/kessoku
```

### Release Management
```bash
# Create a snapshot release (local testing)
go tool goreleaser release --snapshot --clean

# Create a full release (requires git tag)
git tag v1.0.0
go tool goreleaser release --clean
```

## Architecture

### Module Structure
- **Main module**: `github.com/mazrean/kessoku` (Go 1.24)
- **Tools module**: `./tools` - Contains custom linting analyzers and API compatibility checker
- **Go workspace**: Uses go.work with main module and tools

### Code Organization
- `cmd/kessoku/main.go`: Entry point that calls `config.Run()`
- `annotation.go`: Public API (Inject, Provide, Bind, Value, Arg, Async functions) - **Public package root**
- `internal/config/config.go`: CLI configuration and kessoku generation orchestration
- `internal/kessoku/`: Dependency injection implementation
  - `parser.go`: AST parsing for kessoku.Inject calls and provider functions
  - `graph.go`: Dependency graph construction, cycle detection, and async handling
  - `generator.go`: Code generation for injector functions with async support
  - `processor.go`: File processing and orchestration
  - `provider.go`: Core data structures for providers and injectors
  - `const.go`: Package constants
- `internal/pkg/collection/`: Utility data structures
  - `queue.go`: Queue implementation for graph traversal
- `internal/pkg/strings/`: String utility functions
  - `var_name.go`: Variable naming utilities for generated code
- `tools/main.go`: Custom multi-checker with comprehensive Go analyzers
- `examples/`: Example applications demonstrating usage
  - `async_parallel/`: Parallel execution of independent async providers
  - `complex_async/`: Complex async dependency chains with coordination
  - `basic/`: Simple synchronous dependency injection
  - `sets/`: Using value sets for configuration
  - `cross_package/`: Cross-package dependency injection

### Key Dependencies
- `github.com/alecthomas/kong`: CLI argument parsing
- Standard library `log/slog`: Structured logging
- Standard library `go/*`: AST parsing and type checking
- `golang.org/x/tools/go/packages`: Package loading and type information
- `golang.org/x/sync/errgroup`: Async provider coordination (generated code)
- Standard library `context`: Context handling for async operations (generated code)

### Build Configuration
- GoReleaser for cross-platform releases (Linux, Windows, macOS)
- Version injection via ldflags: `version`, `commit`, `date`
- Supports multiple package formats: deb, rpm, apk
- Homebrew tap integration

### Linting Strategy
The tools module provides a comprehensive linting setup combining:
- All govet analyzers
- golangci-lint defaults and optional analyzers  
- staticcheck, simple, and stylecheck analyzers
- Custom multi-checker implementation for unified execution

### Dependency Injection System

Kessoku generates dependency injection code similar to google/wire but with advanced async provider support:

#### Provider Functions
Create provider functions that return dependencies:
```go
// Synchronous provider
func NewConfig() *Config {
    return &Config{Port: 8080}
}

// Async provider for slow operations
func NewDatabase(config *Config) (*Database, error) {
    // Simulate slow database connection
    time.Sleep(200 * time.Millisecond)
    return &Database{URL: config.DatabaseURL}, nil
}

// Async provider without error
func NewCacheService() *CacheService {
    time.Sleep(150 * time.Millisecond)
    return &CacheService{}
}
```

#### Injector Declarations
Use kessoku.Inject to declare dependencies with async support:
```go
package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Async providers execute in parallel
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),                         // Synchronous
    kessoku.Async(kessoku.Provide(NewDatabase)),        // Async parallel
    kessoku.Async(kessoku.Provide(NewCacheService)),    // Async parallel
    kessoku.Provide(NewUserService),                    // Depends on database/cache
    kessoku.Provide(NewApp),                            // Final assembly
)
```

#### Generated Code Features
- **Context Injection**: Automatic `context.Context` parameter when async providers exist
- **Parallel Execution**: Independent async providers run concurrently
- **Channel Synchronization**: Dependent providers wait for async completion
- **Error Handling**: Proper error propagation with cleanup
- **Performance**: Optimized execution paths with minimal overhead

#### Code Generation
Run `go generate` or `go tool kessoku` to generate `*_band.go` files with dependency injection implementations.

Generated async injector signature:
```go
func InitializeApp(ctx context.Context) (*App, error)
```

Generated sync injector signature:
```go
func InitializeApp() (*App, error)
```

## Development Guidelines

### Git Commit Rules
- Always create git commits at appropriate granular units for code changes
- Each commit should represent a logical, atomic change
- Write clear, descriptive commit messages that explain the purpose of the change

### Go Code Quality Rules
- **ALWAYS run lint and test after any Go code changes**
- Run `go tool tools lint ./...` to check for code quality issues
- Run `go test -v ./...` to ensure all tests pass
- Fix any linting errors or test failures before committing
- These checks are mandatory for maintaining code quality standards

### Documentation Maintenance Rules
- **ALWAYS update documentation when making code or feature changes**
- Update CLAUDE.md when:
  - Architecture or module structure changes
  - New commands or development workflows are added
  - Build, test, or deployment processes change
  - Development guidelines or rules are modified
- Update README.md when:
  - User-facing features or functionality change
  - Installation or usage instructions change
  - New command-line options or examples are added
  - Project description or overview needs updating
- Keep documentation in sync with actual implementation
- Documentation updates should be part of the same commit as the related code changes