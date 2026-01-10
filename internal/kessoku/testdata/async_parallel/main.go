package main

import (
	"context"
	"fmt"
)

func main() {
	app, err := InitializeApp(context.Background())
	if err != nil {
		fmt.Println("Error initializing app:", err)
		return
	}

	fmt.Println("App initialized successfully!")
	app.Run()
}
