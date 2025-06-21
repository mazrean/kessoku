# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kessoku is a simple CLI tool built with Go that demonstrates basic CLI patterns with structured logging. It's a greeting application that showcases proper Go project structure using Kong for CLI parsing and slog for structured logging.

## Common Commands

### Building
```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Run directly
go run ./cmd/kessoku [name]
```

### Testing
```bash
# Run tests (note: no test files currently exist)
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
- `internal/config/config.go`: Core application logic with Kong CLI parsing and slog setup
- `tools/main.go`: Custom multi-checker with comprehensive Go analyzers (govet, golangci-lint, staticcheck)

### Key Dependencies
- `github.com/alecthomas/kong`: CLI argument parsing
- Standard library `log/slog`: Structured logging

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