package bind

import (
	"github.com/mazrean/kessoku"
)

var RepoSet = kessoku.Set(
	kessoku.Provide(NewPostgresRepo),
	kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
)
