package main

import (
	"context"
	"fmt"
)

func main() {
	svc, err := InitializeService(context.Background())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	_ = svc
}
