# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kessoku is a dependency injection CLI tool and library for Go, similar to google/wire. It generates Go code for dependency injection based on provider functions and injector declarations. The tool performs compile-time dependency injection through code generation, eliminating runtime reflection overhead.

## Common Commands

### Building
```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Generate dependency injection code (current directory)
go run ./cmd/kessoku

# Generate dependency injection code for specific directory
go run ./cmd/kessoku -d [directory]
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
go run ./tools lint ./...
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
- **Tools module**: `./tools` - Contains custom linting analyzers
- **Go workspace**: Uses go.work with main module and tools

### Code Organization
- `cmd/kessoku/main.go`: Entry point that calls `config.Run()`
- `wire.go`: Public API (Build, NewSet, Bind, Value functions) - **Public package root**
- `internal/config/config.go`: CLI configuration and wire generation orchestration
- `internal/wire/`: Dependency injection implementation
  - `parser.go`: AST parsing for provider functions and build directives
  - `graph.go`: Dependency graph construction and cycle detection
  - `generator.go`: Code generation for injector functions
  - `processor.go`: File processing and orchestration
  - `provider.go`: Core data structures for providers and injectors
- `tools/main.go`: Custom multi-checker with comprehensive Go analyzers
- `examples/`: Example applications demonstrating usage

### Key Dependencies
- `github.com/alecthomas/kong`: CLI argument parsing
- Standard library `log/slog`: Structured logging
- Standard library `go/*`: AST parsing and type checking

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

Kessoku generates dependency injection code similar to google/wire:

#### Provider Functions
Create provider functions that return dependencies:
```go
// NewDatabase creates a database connection.
func NewDatabase(config *Config) (*Database, error) {
    // implementation
}
```

#### Injector Functions
Use kessoku.Build to declare dependencies:
```go
import "github.com/mazrean/kessoku"

func InitializeApp() (*App, error) {
    kessoku.Build(
        NewConfig,
        NewDatabase,
        NewUserService,
        NewApp,
    )
    return nil, nil
}
```

#### Code Generation
Run `kessoku` to generate `*_gen.go` files with dependency injection implementations.

## Development Guidelines

### Git Commit Rules
- Always create git commits at appropriate granular units for code changes
- Each commit should represent a logical, atomic change
- Write clear, descriptive commit messages that explain the purpose of the change

### Go Code Quality Rules
- **ALWAYS run lint and test after any Go code changes**
- Run `go run ./tools lint ./...` to check for code quality issues
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