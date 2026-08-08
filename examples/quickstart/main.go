package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	start := time.Now()
	result := InitApp(context.Background())
	fmt.Printf("%s in %v\n", result, time.Since(start))
}
