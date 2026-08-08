package main

// Service has two string fields to verify separate injection
type Service struct {
	name    string
	address string
}

// NewService takes two string parameters of the same type
func NewService(name string, address string) *Service {
	return &Service{name: name, address: address}
}

func main() {
}
