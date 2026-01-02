//go:generate go tool kessoku $GOFILE

package fieldsof_merge

import (
	"github.com/mazrean/kessoku"
)

var storageSet = kessoku.Set(
	kessoku.Provide(func(s *Storage) (GameImage, GameVideo, GameFile) {
		return s.GameImage, s.GameVideo, s.GameFile
	}),
)
