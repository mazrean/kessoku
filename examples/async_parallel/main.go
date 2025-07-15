package main

import "fmt"

func main() {
	app := InitializeApp()

	fmt.Println("App initialized successfully!")
	app.Run()
}
