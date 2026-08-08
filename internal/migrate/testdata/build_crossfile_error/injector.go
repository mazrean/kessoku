//go:build wireinject

package build_crossfile_error

import "github.com/google/wire"

func InitializeApp() (*App, error) {
	wire.Build(DBSet, NewApp)
	return nil, nil
}
