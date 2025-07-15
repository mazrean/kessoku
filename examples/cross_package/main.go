package main

import (
	"context"
	"fmt"
	
	"github.com/mazrean/kessoku/examples/cross_package/providers"
)

func main() {
	service := InitializeCrossPackageService(context.Background(), providers.APIKey("apiKey"))
	fmt.Printf("Cross-package service initialized: %s\n", service.GetInfo())
}
