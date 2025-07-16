# Development Guide

This document provides guidance for developers working on the Kessoku project.

## Prerequisites

- Go 1.24 or later
- golangci-lint (for linting)

## Building

```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Run directly
go run ./cmd/kessoku ./examples/basic/kessoku.go
```

## Testing

```bash
# Run tests
go test -v ./...

# Format code
go fmt ./...

# Run Go analyzer linter
go tool tools lint ./...

# Test code generation
go generate ./examples/...
```

## Development Workflow

### Code Quality Standards

- **ALWAYS run lint and test after any Go code changes**
- Run `go tool tools lint ./...` to check for code quality issues
- Run `go test -v ./...` to ensure all tests pass
- Fix any linting errors or test failures before committing

### Git Commit Guidelines

- Always create git commits at appropriate granular units for code changes
- Each commit should represent a logical, atomic change
- Write clear, descriptive commit messages that explain the purpose of the change

### Documentation Maintenance

- **ALWAYS update documentation when making code or feature changes**
- Update CLAUDE.md when architecture or module structure changes
- Update README.md when user-facing features or functionality change
- Keep documentation in sync with actual implementation
- Documentation updates should be part of the same commit as the related code changes

## Architecture

### Module Structure
- **Main module**: `github.com/mazrean/kessoku` (Go 1.24)
- **Tools module**: `./tools` - Contains custom linting analyzers and API compatibility checker
- **Go workspace**: Uses go.work with main module and tools

### Code Organization
- `cmd/kessoku/main.go`: Entry point that calls `config.Run()`
- `annotation.go`: Public API (Inject, Provide, Bind, Value, Arg, Async functions)
- `internal/config/config.go`: CLI configuration and kessoku generation orchestration
- `internal/kessoku/`: Dependency injection implementation
  - `parser.go`: AST parsing for kessoku.Inject calls and provider functions
  - `graph.go`: Dependency graph construction, cycle detection, and async handling
  - `generator.go`: Code generation for injector functions with async support
  - `processor.go`: File processing and orchestration
  - `provider.go`: Core data structures for providers and injectors
  - `const.go`: Package constants
- `internal/pkg/collection/`: Utility data structures
- `tools/main.go`: Custom multi-checker with comprehensive Go analyzers
- `examples/`: Example applications demonstrating usage

### Key Features Implementation

#### Async Provider System
- **Parallel Execution**: Independent providers execute concurrently using errgroup
- **Context Injection**: Automatic context.Context injection for async operations
- **Channel Synchronization**: Advanced coordination between dependent async providers
- **Error Handling**: Comprehensive error propagation across async boundaries

#### Code Generation
- **Compile-time Optimization**: All dependency resolution happens at compile time
- **Static Code Generation**: Generates optimized Go code with no reflection
- **Type Safety**: Full compile-time type checking and validation
- **Performance**: Minimal memory allocations and optimal execution paths

### Testing Strategy

- **Unit Tests**: Comprehensive coverage for all core functionality
- **Integration Tests**: End-to-end testing of code generation
- **Async Tests**: Specific testing for parallel execution and context handling
- **Performance Tests**: Ensuring optimal execution paths and memory usage

## API Compatibility

```bash
# Check API compatibility against a previous version
go tool tools apicompat <base_package_path> <target_package_path>

# Example: Check current changes against latest released version
go tool tools apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku

# Example: Check against a specific version
go tool tools apicompat github.com/mazrean/kessoku@v1.0.0 github.com/mazrean/kessoku
```

## Release Management

This project uses GoReleaser for automated releases:

```bash
# Create a snapshot release (local testing)
go tool goreleaser release --snapshot --clean

# Create a full release (requires git tag)
git tag v1.0.0
go tool goreleaser release --clean
```

### Release Process

1. Ensure all tests pass and code quality checks are green
2. Update version numbers and documentation
3. Create a git tag for the release
4. Run GoReleaser to create the release
5. Verify the release artifacts
6. Update package managers (Homebrew, etc.)

## Debugging

### Common Issues

- **Code Generation Failures**: Check AST parsing and type information
- **Async Coordination**: Verify channel synchronization and context handling
- **Memory Leaks**: Ensure proper cleanup of goroutines and channels
- **Performance Issues**: Profile generated code for bottlenecks

### Debugging Tools

- Use `go tool trace` for goroutine analysis
- Use `go tool pprof` for memory and CPU profiling
- Use `go tool compile -d=ssa/help` for SSA debugging
- Use `go build -gcflags="-m"` for escape analysis

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Follow the development workflow and code quality standards
4. Ensure all tests pass and documentation is updated
5. Commit your changes with clear, descriptive messages
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request with a detailed description of changes