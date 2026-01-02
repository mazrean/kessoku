//go:build wireinject

package main

import (
	"github.com/google/wire"
	v1 "github.com/mazrean/kessoku/internal/migrate/testdata/same_name_pkg/api2"
)

var ProductSet = wire.NewSet(v1.NewProduct)
