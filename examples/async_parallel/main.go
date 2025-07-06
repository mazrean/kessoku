package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	// Create a context with timeout for app initialization
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := InitializeApp(ctx)
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}

	fmt.Println("App initialized successfully!")
	app.Run()
}
