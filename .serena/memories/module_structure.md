# Kessoku Module Structure

## Main Module (`github.com/mazrean/kessoku`)
- **Go version**: 1.24+
- **Root files**: Public API (`annotation.go`) with Inject, Provide, Bind, Value, Set functions
- **CLI entry**: `cmd/kessoku/main.go` - calls `config.Run()`

## Directory Structure
```
├── annotation.go              # Public API (Inject, Provide, Bind, Value, Set)
├── cmd/kessoku/              # CLI application entry point
├── internal/
│   ├── config/               # CLI configuration and orchestration
│   ├── kessoku/              # Core DI implementation
│   │   ├── parser.go         # AST parsing for kessoku.Inject calls
│   │   ├── graph.go          # Dependency graph + cycle detection
│   │   ├── generator.go      # Code generation for injectors
│   │   ├── processor.go      # File processing orchestration
│   │   └── provider.go       # Core data structures
│   └── pkg/collection/       # Utility data structures (queue)
├── tools/                    # Custom linting analyzers + API compatibility
├── examples/                 # Usage examples
└── docs/                     # Documentation assets
```

## Tools Module (`./tools`)
- **Custom analyzers**: Comprehensive Go linting with govet, golangci-lint, staticcheck
- **API compatibility**: Version compatibility checking tool
- **Build integration**: GoReleaser configuration for releases

## Workspace Configuration
- Uses `go.work` with main module and tools module
- Separate dependency management for development tools