package main

import (
	"context"
	"fmt"
)

func main() {
	app := InitializeComplexApp(context.Background())

	fmt.Println("Complex app initialized successfully!")
	app.Run()
}
