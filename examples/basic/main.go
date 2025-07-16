package main

import (
	"fmt"
	"log"
)

func main() {
	app, err := InitializeApp()
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}

	fmt.Println("App initialized successfully!")
	app.Run()
}