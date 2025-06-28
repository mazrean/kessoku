package main

import (
	"fmt"

	"github.com/mazrean/kessoku/examples/cross_package/providers"
)

func main() {
	service := InitializeCrossPackageService(providers.APIKey("apiKey"))
	fmt.Printf("Cross-package service initialized: %s\n", service.GetInfo())
}
