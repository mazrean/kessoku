package build

import (
	"github.com/mazrean/kessoku"
)

func InitializeApp() (*App, error) {
	return kessoku.Inject[*App](kessoku.Provide(NewDB), kessoku.Provide(NewApp))
}
