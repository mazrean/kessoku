# Interface Binding Example

This example demonstrates how to use `kessoku.As` and `kessoku.AsMap` for interface binding in dependency injection.

## Features Demonstrated

1. **kessoku.As**: Binds concrete implementations to interfaces
   - `DatabaseUserRepository` → `UserRepository`
   - `ConsoleLogger` → `Logger`

2. **kessoku.AsMap**: Explicit source-to-destination type mapping
   - `*ConsoleLogger` → `Logger` with explicit type safety
   - More verbose than `As` but provides clearer type relationships

## Usage

```bash
# Generate dependency injection code
go tool kessoku kessoku.go

# Run the example
go run .
```

## Key Concepts

- `kessoku.As[Interface]` replaces the older `kessoku.Bind[Interface]` syntax
- `kessoku.AsMap[DestinationType, SourceType]` provides explicit type mapping with better type safety
- The generated `*_band.go` file contains the actual implementation code

## Generated Functions

- `InitializeUserService() *UserService` - demonstrates `kessoku.As`
- `InitializeEmailService() *EmailNotificationService` - demonstrates `kessoku.AsMap`

## Differences between As and AsMap

- **As**: `kessoku.As[Interface](provider)` - simpler syntax for interface binding
- **AsMap**: `kessoku.AsMap[Interface, ConcreteType](provider)` - explicit source and destination types for better type safety and clarity