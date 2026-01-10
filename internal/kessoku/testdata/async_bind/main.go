package main

import "context"

type Repository interface {
	Find(ctx context.Context, id string) (string, error)
}

type DatabaseRepo struct{}

func NewDatabaseRepo() *DatabaseRepo {
	return &DatabaseRepo{}
}

func (r *DatabaseRepo) Find(ctx context.Context, id string) (string, error) {
	return "data-" + id, nil
}

type Service struct {
	repo *DatabaseRepo
}

func NewService(repo *DatabaseRepo) *Service {
	return &Service{repo: repo}
}

func main() {
}
