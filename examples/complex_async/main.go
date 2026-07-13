package main

import "fmt"

func main() {
	app := InitializeComplexApp()

	fmt.Println("Complex app initialized successfully!")
	app.Run()
}
