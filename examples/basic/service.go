package main

import (
	"fmt"
	"log"
)

// UserService provides user-related operations.
type UserService struct {
	db     *Database
	config *Config
}

// NewUserService creates a new user service.
func NewUserService(db *Database, config *Config) *UserService {
	if config.Debug {
		log.Println("Creating user service")
	}
	
	return &UserService{
		db:     db,
		config: config,
	}
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(id int) (*User, error) {
	if s.config.Debug {
		log.Printf("UserService: Getting user %d", id)
	}
	
	user, err := s.db.GetUser(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// ListUsers returns all users.
func (s *UserService) ListUsers() []*User {
	if s.config.Debug {
		log.Println("UserService: Listing all users")
	}
	
	users := []*User{}
	for i := 1; i <= 10; i++ {
		if user, err := s.db.GetUser(i); err == nil {
			users = append(users, user)
		}
	}
	
	return users
}