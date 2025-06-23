// Package main demonstrates various ways to use kessoku.Set for organizing providers.
package main

import (
	"fmt"
	"log"
)

func main() {
	// Example 1: Initialize using basic inline Set
	fmt.Println("=== Example 1: Inline Set ===")
	app1, err := InitializeApp()
	if err != nil {
		log.Fatal(err)
	}
	app1.Run()

	fmt.Println()

	// Example 2: Initialize using Set variables
	fmt.Println("=== Example 2: Set Variables ===")
	app2, err := InitializeAppWithSets()
	if err != nil {
		log.Fatal(err)
	}
	app2.Run()

	fmt.Println()

	// Example 3: Initialize just a service
	fmt.Println("=== Example 3: Service Only ===")
	userService, err := InitializeUserService()
	if err != nil {
		log.Fatal(err)
	}
	user, _ := userService.GetUser("123")
	fmt.Printf("Retrieved user: %+v\n", user)

	fmt.Println()

	// Example 4: Initialize using nested Sets
	fmt.Println("=== Example 4: Nested Sets ===")
	app4, err := InitializeAppWithNestedSets()
	if err != nil {
		log.Fatal(err)
	}
	app4.Run()
}

// App represents the main application
type App struct {
	server  *Server
	service *UserService
}

// Run starts the application
func (a *App) Run() {
	fmt.Printf("Starting application with server on %s\n", a.server.config.Port)
	fmt.Printf("User service connected to database: %s\n", a.service.db.connectionString)
	fmt.Println("Application running successfully!")
}

// NewApp creates a new application instance
func NewApp(server *Server, service *UserService) *App {
	return &App{
		server:  server,
		service: service,
	}
}