package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// Create a context with timeout for app initialization
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app := InitializeApp(ctx)

	fmt.Println("App initialized successfully!")
	app.Run()
}
