package main

type DB struct{}

func NewDB() *DB {
	return &DB{}
}

func ValidateDB(db *DB) error {
	return nil
}

type Service struct {
	db *DB
}

func NewService(db *DB) *Service {
	return &Service{db: db}
}

func main() {
}
