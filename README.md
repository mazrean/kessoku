# Kessoku

A dependency injection code generator for Go, similar to [google/wire](https://github.com/google/wire). Kessoku generates Go code for compile-time dependency injection, eliminating runtime reflection overhead.

## Features

- ✅ **Compile-time dependency injection** - No runtime reflection
- ✅ **Automatic dependency resolution** - Topological sorting of dependencies  
- ✅ **Error handling** - Proper error propagation in generated code
- ✅ **Cycle detection** - Prevents circular dependencies
- ✅ **Wire compatibility** - Similar API to google/wire
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

```go
// InitializeApp creates the application with all dependencies.
func InitializeApp() (*App, error) {
    kessoku.Build(
        NewConfig,
        NewDatabase,
        NewUserService,
        NewApp,
    )
    return nil, nil // This will be replaced by generated code
}
```

### 3. Generate Dependency Injection Code

```bash
kessoku  # Generate code in current directory
# or
kessoku -d ./path/to/your/package
```

This generates `*_gen.go` files with the actual dependency injection implementation.

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
# Generate DI code in current directory
kessoku

# Generate DI code in specific directory
kessoku -d ./path/to/package

# Show version
kessoku --version
```

### Options

- `-d, --dir` - Directory to process (defaults to current directory)
- `-l, --log-level` - Log level (debug, info, warn, error)
- `--version` - Show version information

## API Reference

### kessoku.Build

Declares the dependencies needed for an injector function:

```go
func InitializeApp() (*App, error) {
    kessoku.Build(
        NewConfig,    // Provider functions
        NewDatabase,
        NewService,
        NewApp,
    )
    return nil, nil
}
```

### kessoku.NewSet

Groups related providers into a set:

```go
var DatabaseSet = kessoku.NewSet(
    NewConfig,
    NewDatabase,
)

func InitializeApp() (*App, error) {
    kessoku.Build(
        DatabaseSet,
        NewService,
        NewApp,
    )
    return nil, nil
}
```

### kessoku.Bind

Binds an interface to its implementation:

```go
var UserRepositoryBinding = kessoku.Bind(new(UserRepository), new(*userRepositoryImpl))
```

### kessoku.Value

Provides a constant value:

```go
var DatabaseURL = kessoku.Value("postgres://localhost/mydb")
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
go run ./cmd/kessoku -d ./examples/basic
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
