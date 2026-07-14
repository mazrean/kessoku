package impl

// Querier is the interface for database queries.
type Querier interface {
	Query(q string) (string, error)
}

// DB is the concrete implementation.
type DB struct{}

// Query implements Querier.
func (d *DB) Query(q string) (string, error) {
	return "", nil
}
