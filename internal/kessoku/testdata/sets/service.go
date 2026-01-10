package main

// UserService provides user-related business logic
type UserService struct {
	db *Database
}

// NewUserService creates a new user service
func NewUserService(db *Database) *UserService {
	return &UserService{
		db: db,
	}
}
