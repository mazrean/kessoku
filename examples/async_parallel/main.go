package main

import "fmt"

func main() {
	app, err := InitializeApp()
	if err != nil {
		fmt.Println("Error initializing app:", err)
		return
	}

	fmt.Println("App initialized successfully!")
	app.Run()
}
