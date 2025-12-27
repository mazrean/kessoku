# Suggested Commands for Kessoku Development

## Build and Development Commands
```bash
# Build the binary
go build -o bin/kessoku ./cmd/kessoku

# Generate dependency injection code using go generate
go generate ./...

# Generate dependency injection code directly
go tool kessoku [files...]
```

## Testing and Quality Commands
```bash
# Run tests
go test -v ./...

# Format code
go fmt ./...

# Run comprehensive linting
go tool tools lint ./...
```

## API Compatibility Checking
```bash
# Check API compatibility against a previous version
go tool tools apicompat <base_package_path> <target_package_path>

# Example: Check current changes against latest released version
go tool tools apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku

# Example: Check against a specific version
go tool tools apicompat github.com/mazrean/kessoku@v1.0.0 github.com/mazrean/kessoku
```

## Release Management
```bash
# Create a snapshot release (local testing)
go tool goreleaser release --snapshot --clean

# Create a full release (requires git tag)
git tag v1.0.0
go tool goreleaser release --clean
```

## System Utilities (Linux)
- `git` - version control
- `ls` - list files/directories
- `cd` - change directory
- `grep`/`rg` - search text patterns
- `find` - find files
- `cat` - display file contents