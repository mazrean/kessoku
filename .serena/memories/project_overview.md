# Kessoku Project Overview

## Purpose
Kessoku is a compile-time dependency injection library for Go, similar to google/wire but with parallel execution capabilities. It generates optimized Go code for dependency injection that can run independent providers concurrently, reducing application startup time by up to 2.25x.

## Tech Stack
- **Language**: Go (1.24+)
- **CLI Framework**: github.com/alecthomas/kong
- **Build Tool**: GoReleaser for cross-platform releases
- **Workspace**: Uses go.work with main module and tools module
- **Logging**: Standard library log/slog for structured logging
- **AST Processing**: Standard library go/* packages for parsing and type checking

## Key Features
- Compile-time dependency injection (no runtime reflection)
- Automatic parallel execution of independent providers
- Compatible with google/wire patterns but with performance improvements
- Code generation via `go tool kessoku` command
- Support for async providers, sets, value injection, and interface binding