package main

// DB is the concrete database struct.
type DB struct{ connected bool }

// DBAlias is a type alias for *DB.
type DBAlias = *DB

// NewDB creates a DB instance, returning the alias type so that the alias-keyed
// dependency for DBAlias is satisfied without an extra injector argument.
// After the alias-key fix, each alias gets its own dependency key, so a
// provider must declare the alias return type to satisfy alias-typed parameters.
func NewDB() DBAlias {
	return &DB{connected: true}
}

// ValidateDB is an error-only validator provider that takes a DBAlias parameter.
func ValidateDB(db DBAlias) error {
	return nil
}

// Service depends on DBAlias (*DB).
type Service struct{}

// NewService creates a Service.
func NewService(db DBAlias) *Service {
	return &Service{}
}

func main() {}
