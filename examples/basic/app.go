package main

import (
	"fmt"
	"log"
)

// App represents the main application.
type App struct {
	config      *Config
	userService *UserService
}

// NewApp creates a new application instance.
func NewApp(config *Config, userService *UserService) *App {
	if config.Debug {
		log.Println("Creating application")
	}
	
	return &App{
		config:      config,
		userService: userService,
	}
}

// Run starts the application.
func (a *App) Run() {
	fmt.Printf("Starting %s\n", a.config.AppName)
	
	// Demonstrate the app functionality
	users := a.userService.ListUsers()
	fmt.Printf("Found %d users:\n", len(users))
	
	for _, user := range users {
		fmt.Printf("  - %s (%s)\n", user.Name, user.Email)
	}
	
	// Get a specific user
	if user, err := a.userService.GetUser(1); err == nil {
		fmt.Printf("\nUser 1 details: %+v\n", user)
	} else {
		fmt.Printf("\nError getting user 1: %v\n", err)
	}
	
	fmt.Println("\nApplication completed successfully!")
}