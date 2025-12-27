# Kessoku Coding Style and Conventions

## Go Code Style
- **Standard Go conventions**: Follow `go fmt` formatting
- **Package organization**: Clear separation between public API (root) and internal implementation
- **Error handling**: Standard Go error handling patterns with explicit error returns
- **Logging**: Use structured logging with `log/slog` package
- **Generics**: Used extensively for type-safe dependency injection (Go 1.24+)

## Naming Conventions
- **File naming**: Snake_case for generated files (`*_band.go`)
- **Package structure**: `internal/` for implementation details, root for public API
- **Variable naming**: Standard Go camelCase
- **Constants**: ALL_CAPS for package constants

## Code Organization Patterns
- **Interface design**: Marker interfaces (like `provider`) for compile-time type checking
- **AST processing**: Extensive use of Go's `go/ast` and `go/types` packages
- **Code generation**: Template-based generation in `generator.go`
- **Dependency graphs**: Graph-based dependency resolution with cycle detection

## Documentation
- **Package docs**: Clear package-level documentation explaining purpose
- **Function docs**: Comprehensive examples in public API functions
- **Code comments**: Explain complex algorithms (especially in graph and generator logic)

## Testing Patterns
- **Unit tests**: Comprehensive test coverage with `*_test.go` files
- **Test helpers**: Shared test utilities for creating test data
- **Benchmarks**: Performance benchmarks for critical paths