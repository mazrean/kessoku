# Kessoku

A dependency injection code generator for Go, similar to [google/wire](https://github.com/google/wire). Kessoku generates Go code for compile-time dependency injection, eliminating runtime reflection overhead.

## Features

- ✅ **Compile-time dependency injection** - No runtime reflection
- ✅ **Automatic dependency resolution** - Topological sorting of dependencies  
- ✅ **Error handling** - Proper error propagation in generated code
- ✅ **Cycle detection** - Prevents circular dependencies
- ✅ **Go generate integration** - Works seamlessly with `go generate`
- ✅ **Cross-platform support** (Linux, Windows, macOS)

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

## Quick Start

### 1. Create Provider Functions

```go
package main

import "github.com/mazrean/kessoku"

type Config struct {
    DatabaseURL string
    Port        int
}

func NewConfig() *Config {
    return &Config{
        DatabaseURL: "postgres://localhost/mydb",
        Port:        8080,
    }
}

type Database struct {
    url string
}

func NewDatabase(config *Config) (*Database, error) {
    return &Database{url: config.DatabaseURL}, nil
}

type UserService struct {
    db *Database
}

func NewUserService(db *Database) *UserService {
    return &UserService{db: db}
}

type App struct {
    config  *Config
    service *UserService
}

func NewApp(config *Config, service *UserService) *App {
    return &App{
        config:  config,
        service: service,
    }
}
```

### 2. Define Injector Function

Create a file `kessoku.go` with injector declarations:

```go
package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// InitializeApp creates the application with all dependencies.
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDatabase),
    kessoku.Provide(NewUserService),
    kessoku.Provide(NewApp),
)
```

### 3. Generate Dependency Injection Code

```bash
# Using go generate (recommended)
go generate ./...

# Or run kessoku directly
go tool kessoku kessoku.go
```

This generates `*_band.go` files with the actual dependency injection implementation.

### 4. Use the Generated Code

```go
func main() {
    app, err := InitializeApp()
    if err != nil {
        log.Fatal("Failed to initialize app:", err)
    }
    
    // Use your app
    fmt.Printf("App running on port %d\n", app.config.Port)
}
```

## CLI Usage

```bash
# Generate DI code for specific files
go tool kessoku kessoku.go

# Multiple files
go tool kessoku file1.go file2.go

# Using go generate (recommended)
go generate ./...

# Show version
go tool kessoku --version
```

### Options

- `-l, --log-level` - Log level (debug, info, warn, error)
- `-v, --version` - Show version information

## API Reference

### kessoku.Inject

Declares an injector function with its dependencies:

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",       // Function name to generate
    kessoku.Provide(NewConfig),    // Provider functions
    kessoku.Provide(NewDatabase),
    kessoku.Provide(NewService),
    kessoku.Provide(NewApp),
)
```

### kessoku.Provide

Wraps a provider function for dependency injection:

```go
kessoku.Provide(NewConfig)     // Provides *Config
kessoku.Provide(NewDatabase)   // Provides *Database, error
```

### kessoku.Bind

Binds an interface to its implementation:

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Bind[UserRepository](kessoku.Provide(NewUserRepositoryImpl)),
    kessoku.Provide(NewApp),
)
```

### kessoku.Value

Provides a constant value:

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Value("postgres://localhost/mydb"),  // Provides string
    kessoku.Provide(NewApp),
)
```

### kessoku.Arg

Declares a runtime argument to be passed to the injector:

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Arg[*Config]("config"),  // Runtime argument
    kessoku.Provide(NewApp),
)
```

## Examples

See the [examples/](./examples/) directory for complete working examples.

## Development

### Prerequisites

- Go 1.24 or later
- golangci-lint (for linting)

### Building

```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Run directly
go run ./cmd/kessoku ./examples/basic/kessoku.go
```

### Testing

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
