package bind

import "github.com/google/wire"

type Repository interface {
	Get() string
}

type PostgresRepo struct{}

func (p *PostgresRepo) Get() string {
	return "postgres"
}

func NewPostgresRepo() *PostgresRepo {
	return &PostgresRepo{}
}

var RepoSet = wire.NewSet(
	NewPostgresRepo,
	wire.Bind(new(Repository), new(*PostgresRepo)),
)
