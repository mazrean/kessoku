# Sets Example

This example demonstrates how to use `kessoku.Set` to organize and group providers for cleaner dependency injection.

## What is kessoku.Set?

`kessoku.Set` allows you to group related providers together, making your dependency injection setup more organized and reusable.

## Running the Example

```bash
# Generate dependency injection code
go generate ./examples/sets

# Run the example
go run ./examples/sets
```

## Examples

### 1. Basic usage (without Sets)

```go
var _ = kessoku.Inject[*App](
    "InitializeAppBasic",
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDatabase),
    kessoku.Provide(NewUserService),
    kessoku.Provide(NewApp),
)
```

### 2. Using inline kessoku.Set

Group related providers together:

```go
var _ = kessoku.Inject[*App](
    "InitializeAppWithInlineSet",
    kessoku.Set(
        kessoku.Provide(NewConfig),
        kessoku.Provide(NewDatabase),
    ),
    kessoku.Provide(NewUserService),
    kessoku.Provide(NewApp),
)
```

### 3. Using Set variables

Define reusable Sets:

```go
var DatabaseSet = kessoku.Set(
    kessoku.Provide(NewConfig),
    kessoku.Provide(NewDatabase),
)

var _ = kessoku.Inject[*App](
    "InitializeAppWithSetVariable",
    DatabaseSet, // Reuse the set
    kessoku.Provide(NewUserService),
    kessoku.Provide(NewApp),
)
```

### 4. Nested Sets

Use Sets inside other Sets:

```go
var ServiceSet = kessoku.Set(
    DatabaseSet, // Include another set
    kessoku.Provide(NewUserService),
)

var _ = kessoku.Inject[*App](
    "InitializeAppWithNestedSets",
    ServiceSet, // This includes both database and service
    kessoku.Provide(NewApp),
)
```

## Benefits

- **Organization**: Group related providers logically
- **Reusability**: Define once, use multiple times
- **Readability**: Cleaner injection setup
- **Modularity**: Separate concerns into different Sets

## Files

- `main.go` - Main application demonstrating all examples
- `config.go` - Database configuration
- `database.go` - Database connection
- `service.go` - User service
- `kessoku.go` - Dependency injection setup with Sets
- `kessoku_band.go` - Generated code