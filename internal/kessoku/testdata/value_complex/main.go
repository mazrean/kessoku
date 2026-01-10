package main

type Service struct {
	config map[string]string
	tags   []string
}

func NewService(config map[string]string, tags []string) *Service {
	return &Service{config: config, tags: tags}
}

func main() {
}
