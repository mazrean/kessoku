package main

import (
	"fmt"
	"log"
)

//go:generate go tool kessoku kessoku.go

// Interface definitions
type UserRepository interface {
	GetUser(id string) (*User, error)
}

type Logger interface {
	Log(message string)
}

type NotificationService interface {
	Send(message string) error
}

// Data types
type User struct {
	ID   string
	Name string
}

// Concrete implementations
type DatabaseUserRepository struct{}

func (r *DatabaseUserRepository) GetUser(id string) (*User, error) {
	return &User{ID: id, Name: "User " + id}, nil
}

func NewDatabaseUserRepository() *DatabaseUserRepository {
	return &DatabaseUserRepository{}
}

type ConsoleLogger struct{}

func (l *ConsoleLogger) Log(message string) {
	fmt.Printf("[LOG] %s\n", message)
}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

type EmailNotificationService struct {
	logger Logger
}

func (s *EmailNotificationService) Send(message string) error {
	s.logger.Log("Sending email: " + message)
	return nil
}

func NewEmailNotificationService(logger Logger) *EmailNotificationService {
	return &EmailNotificationService{logger: logger}
}

type SMSNotificationService struct {
	logger Logger
}

func (s *SMSNotificationService) Send(message string) error {
	s.logger.Log("Sending SMS: " + message)
	return nil
}

func NewSMSNotificationService(logger Logger) *SMSNotificationService {
	return &SMSNotificationService{logger: logger}
}

// Services that use interfaces
type UserService struct {
	repo   UserRepository
	logger Logger
}

func NewUserService(repo UserRepository, logger Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

func (s *UserService) GetUser(id string) (*User, error) {
	s.logger.Log("Getting user: " + id)
	return s.repo.GetUser(id)
}

type NotificationManager struct {
	services map[string]NotificationService
}

func NewNotificationManager(services map[string]NotificationService) *NotificationManager {
	return &NotificationManager{services: services}
}

func (m *NotificationManager) SendAll(message string) {
	for name, service := range m.services {
		fmt.Printf("Using %s service:\n", name)
		if err := service.Send(message); err != nil {
			log.Printf("Failed to send via %s: %v", name, err)
		}
	}
}

func main() {
	// Use the generated injector function to demonstrate kessoku.As functionality
	userService := InitializeUserService()

	user, err := userService.GetUser("123")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved user: %+v\n", user)
	fmt.Println("Successfully demonstrated kessoku.As for interface binding!")
}