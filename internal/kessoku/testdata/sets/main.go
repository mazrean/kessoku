// Package main demonstrates various ways to use kessoku.Set for organizing providers.
package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("kessoku.Set Examples")
	fmt.Println("====================")
	fmt.Println()

	// Example 1: Basic usage without Sets (for comparison)
	fmt.Println("1. Basic usage (without Sets):")
	app1, err := InitializeAppBasic()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ App initialized with database: %s\n", app1.service.db.connectionString)
	fmt.Println()

	// Example 2: Using inline Set
	fmt.Println("2. Using inline kessoku.Set:")
	_, err = InitializeAppWithInlineSet()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ App initialized with grouped providers\n")
	fmt.Println()

	// Example 3: Using Set variable
	fmt.Println("3. Using Set variable for reusability:")
	_, err = InitializeAppWithSetVariable()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ App initialized using DatabaseSet variable\n")
	fmt.Println()

	// Example 4: Nested Sets
	fmt.Println("4. Using nested Sets:")
	_, err = InitializeAppWithNestedSets()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ App initialized using ServiceSet (which includes DatabaseSet)\n")
	fmt.Println()

	fmt.Println("All examples completed successfully!")
}

// App represents the main application
type App struct {
	service *UserService
}

// NewApp creates a new application instance
func NewApp(service *UserService) *App {
	return &App{
		service: service,
	}
}
