package main

import (
	"github.com/mazrean/kessoku"
	v1 "github.com/mazrean/kessoku/internal/migrate/testdata/same_name_pkg/api1"
	v1_1 "github.com/mazrean/kessoku/internal/migrate/testdata/same_name_pkg/api2"
)

var UserSet = kessoku.Set(
	kessoku.Provide(v1.NewUser),
)
var ProductSet = kessoku.Set(
	kessoku.Provide(v1_1.NewProduct),
)
