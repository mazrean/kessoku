package main

// Errgroup is a user-defined type whose lowerCamel base name collides with
// the "golang.org/x/sync/errgroup" import alias that kessoku injects.
type Errgroup struct {
	Value int
}

type Database struct{}

type Service struct{}

func NewErrgroup() (*Errgroup, error) {
	return &Errgroup{Value: 42}, nil
}

func NewDatabase() (*Database, error) {
	return &Database{}, nil
}

func NewService(eg *Errgroup, db *Database) *Service {
	return &Service{}
}
