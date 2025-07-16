package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("🚀 Kessoku Async Parallel Example")
	fmt.Println("==================================")
	
	start := time.Now()
	
	// kessoku.Async() enables parallel execution
	app, err := InitializeApp(context.Background())
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}
	
	duration := time.Since(start)
	
	fmt.Printf("\n⏱️  Total initialization time: %v\n", duration)
	fmt.Printf("💡 Without kessoku.Async(): ~450ms (200+150+100)\n")
	fmt.Printf("⚡ With kessoku.Async(): ~200ms (max of 200,150,100)\n")
	fmt.Printf("🎯 Performance improvement: %.1fx faster!\n\n", 450.0/float64(duration.Milliseconds()))
	
	app.Run()
}