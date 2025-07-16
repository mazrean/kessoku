package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("🎯 Kessoku Sets Example")
	fmt.Println("=======================")
	fmt.Println()

	// Example 1: Basic Set
	fmt.Println("1. Basic Set (grouped providers):")
	app1, err := InitializeAppBasic()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✅ App initialized with database: %s\n", app1.service.db.connectionString)
	fmt.Println()

	// Example 2: Reusable Set
	fmt.Println("2. Reusable Set (DatabaseSet variable):")
	app2, err := InitializeAppWithSet()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✅ App initialized using DatabaseSet: %s\n", app2.service.db.connectionString)
	fmt.Println()

	// Example 3: Nested Sets
	fmt.Println("3. Nested Sets (ServiceSet includes DatabaseSet):")
	app3, err := InitializeAppWithNestedSets()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✅ App initialized using ServiceSet: %s\n", app3.service.db.connectionString)
	fmt.Println()

	fmt.Println("🎉 All examples completed successfully!")
	fmt.Println()
	fmt.Println("💡 Key benefits of kessoku.Set:")
	fmt.Println("   • Organization: Group related providers logically")
	fmt.Println("   • Reusability: Define once, use multiple times")
	fmt.Println("   • Modularity: Separate concerns into different sets")
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