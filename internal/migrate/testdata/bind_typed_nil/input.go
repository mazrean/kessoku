//go:build wireinject

package bind_typed_nil

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

// Typed-nil form of wire.Bind: some codebases use (*T)(nil) instead of new(T).
// This form cannot be resolved to a type via extractTypeFromNew and must be
// silently skipped rather than panicking.
var RepoSet = wire.NewSet(
	NewPostgresRepo,
	wire.Bind((*Repository)(nil), (*PostgresRepo)(nil)),
)
