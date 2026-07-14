package main

import "github.com/mazrean/kessoku/internal/kessoku/testdata/dot_import_set/setpkg"

// Service represents an application service.
type Service struct {
	config *setpkg.Config
}

// NewService creates a new Service.
func NewService(config *setpkg.Config) *Service {
	return &Service{config: config}
}

func main() {}
