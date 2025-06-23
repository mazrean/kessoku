package main

import "fmt"

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

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id string) (*User, error) {
	fmt.Printf("Getting user %s from database\n", id)
	return &User{
		ID:   id,
		Name: "John Doe",
	}, nil
}

// User represents a user entity
type User struct {
	ID   string
	Name string
}