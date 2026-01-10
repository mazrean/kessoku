package main

type Repository interface {
	Find(id string) string
}

type memoryRepo struct{}

func (r *memoryRepo) Find(id string) string {
	return "data-" + id
}

// NewRepository returns interface type directly
func NewRepository() Repository {
	return &memoryRepo{}
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func main() {
}
