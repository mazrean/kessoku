package main

import (
	"fmt"
	"os"

	"github.com/mazrean/kessoku/internal/config"
)

func main() {
	if err := config.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
