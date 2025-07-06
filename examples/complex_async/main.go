package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	// Create a context with timeout for app initialization
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	app, err := InitializeComplexApp(ctx)
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}

	fmt.Println("Complex app initialized successfully!")
	app.Run()
}