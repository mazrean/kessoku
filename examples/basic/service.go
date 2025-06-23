package main

import (
	"fmt"
	"log/slog"
)

// UserService provides user-related operations.
type UserService struct {
	db     *Database
	logger *slog.Logger
}

// NewUserService creates a new user service.
// wire: provider
func NewUserService(db *Database, logger *slog.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(id int) (*User, error) {
	s.logger.Info("Getting user", "id", id)

	rows, err := s.db.Query("SELECT id, name, email FROM users WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error("Failed to close rows", "error", closeErr)
		}
	}()

	if !rows.Next() {
		return nil, fmt.Errorf("user not found")
	}

	var user User
	if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return &user, nil
}

// User represents a user entity.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	ID    int    `json:"id"`
}

// NewLogger creates a new logger instance.
// wire: provider
func NewLogger() *slog.Logger {
	return slog.Default()
}
