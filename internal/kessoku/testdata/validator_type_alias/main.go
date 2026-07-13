package main

// DB is the concrete database struct.
type DB struct{ connected bool }

// DBAlias is a type alias for *DB.
type DBAlias = *DB

// NewDB creates a DB instance.
func NewDB() *DB {
	return &DB{connected: true}
}

// ValidateDB is an error-only validator provider that takes a DBAlias parameter.
// Before the QA-3 fix, the validator third pass used t.String() to look up
// fnProviderMap, which returned "main.DBAlias" for the alias type while the
// map was keyed with typeKey(t) = "*main.DB". This caused the lookup to miss
// and autoAddMissingDependencies to create a spurious extra injector argument.
func ValidateDB(db DBAlias) error {
	return nil
}

// Service depends on *DB.
type Service struct{}

// NewService creates a Service.
func NewService(db *DB) *Service {
	return &Service{}
}

func main() {}
