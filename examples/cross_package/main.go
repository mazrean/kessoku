package main

import (
	"fmt"
)

func main() {
	service := InitializeCrossPackageService()
	fmt.Printf("Cross-package service initialized: %s\n", service.GetInfo())
}
