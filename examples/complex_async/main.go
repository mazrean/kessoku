package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// Create a context with timeout for app initialization
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	app, err := InitializeComplexApp(ctx)
	if err != nil {
		fmt.Println("Failed to initialize complex app:", err)
		return
	}

	fmt.Println("Complex app initialized successfully!")
	app.Run()
}
