# kessoku

A simple CLI tool built with Go and Kong that demonstrates basic CLI patterns with structured logging.

## Features

- Simple greeting functionality
- Command-line argument parsing with Kong
- Structured logging with slog
- Cross-platform support (Linux, Windows, macOS)
- Version information injection at build time

## Installation

### From Source

```bash
go install github.com/mazrean/kessoku/cmd/kessoku@latest
```

### From Releases

Download the latest binary from the [releases page](https://github.com/mazrean/kessoku/releases).

### Via Homebrew

```bash
brew install mazrean/tap/kessoku
```

## Usage

Basic usage:

```bash
# Default greeting
kessoku

# Greet a specific name
kessoku Alice

# Show version
kessoku --version
```

### Options

- `-l, --log-level` - Log level (debug, info, warn, error)
- `--version` - Show version information
- `[name]` - Optional name to greet (defaults to "World")

## Development

### Prerequisites

- Go 1.24 or later
- golangci-lint (for linting)

### Building

```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Run directly
go run ./cmd/kessoku [name]
```

### Testing

```bash
# Run tests
go test -v ./...

# Format code
go fmt ./...

# Run Go analyzer linter
go run ./tools lint ./...
```

### Releasing

This project uses GoReleaser for automated releases:

```bash
# Create a snapshot release (local testing)
go tool goreleaser release --snapshot --clean

# Create a full release (requires git tag)
git tag v1.0.0
go tool goreleaser release --clean
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
