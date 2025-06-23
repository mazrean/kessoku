# Sets Example

This example demonstrates various ways to use `kessoku.Set` to organize and group providers for better dependency injection management.

## Overview

The `kessoku.Set` function allows you to group related providers together, making your dependency injection declarations more organized and reusable. This example shows four different approaches:

1. **Inline Sets**: Using `kessoku.Set` directly within `kessoku.Inject`
2. **Set Variables**: Pre-defining Sets as variables for reuse
3. **Nested Sets**: Using Sets within other Sets
4. **Mixed Usage**: Combining Sets with individual providers

## Files

- `main.go` - Main application entry point that demonstrates all approaches
- `config.go` - Configuration management
- `database.go` - Database connection management
- `server.go` - HTTP server implementation
- `service.go` - Business logic services
- `kessoku.go` - Dependency injection setup using various Set patterns
- `kessoku_band.go` - Generated dependency injection code

## Running the Example

```bash
# Generate dependency injection code
go generate ./examples/sets

# Run the example
go run ./examples/sets
```

## Set Usage Patterns

### 1. Inline Sets

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    // Inline Set usage - groups infrastructure providers
    kessoku.Set(
        kessoku.Provide(NewConfig),
        kessoku.Provide(NewDatabase),
        kessoku.Provide(NewServer),
    ),
    kessoku.Provide(NewUserService),
    kessoku.Provide(NewApp),
)
```

### 2. Set Variables

```go
// Define reusable Sets
var DatabaseSet = kessoku.Set(
    kessoku.Provide(NewDatabase),
)

var ServerSet = kessoku.Set(
    kessoku.Provide(NewServer),
)

// Use Sets as variables
var _ = kessoku.Inject[*App](
    "InitializeAppWithSets",
    kessoku.Provide(NewConfig),
    ServerSet,  // Use pre-defined server set
    ServiceSet, // Use pre-defined service set
    kessoku.Provide(NewApp),
)
```

### 3. Nested Sets

```go
// Sets can contain other Sets
var ServiceSet = kessoku.Set(
    DatabaseSet, // Use another set within a set
    kessoku.Provide(NewUserService),
)

var CoreInfrastructureSet = kessoku.Set(
    kessoku.Set( // Nested inline Set
        kessoku.Provide(NewDatabase),
        kessoku.Provide(NewServer),
    ),
)
```

### 4. Mixed Usage

You can freely mix inline Sets, Set variables, and individual providers:

```go
var _ = kessoku.Inject[*App](
    "InitializeApp",
    MyInfrastructureSet,     // Set variable
    kessoku.Set(             // Inline Set
        kessoku.Provide(NewService1),
        kessoku.Provide(NewService2),
    ),
    kessoku.Provide(NewApp), // Individual provider
)
```

## Benefits of Using Sets

1. **Organization**: Group related providers together logically
2. **Reusability**: Define Sets once and reuse them across multiple injectors
3. **Maintainability**: Easier to manage complex dependency graphs
4. **Modularity**: Create modular provider groups for different application layers
5. **Readability**: Cleaner and more understandable dependency injection setup

## Generated Code

All Set declarations are flattened during code generation, so the final generated functions contain the optimal dependency resolution order regardless of how you organize your Sets.