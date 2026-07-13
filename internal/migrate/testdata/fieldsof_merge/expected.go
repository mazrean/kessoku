//go:build !wireinject

//go:generate go tool kessoku $GOFILE

package fieldsof_merge

import (
	"github.com/mazrean/kessoku"
)

var storageSet = kessoku.Set(
	kessoku.Provide(func(s *Storage) (GameImage, *GameImage, GameVideo, *GameVideo, GameFile, *GameFile) {
		return s.GameImage, &s.GameImage, s.GameVideo, &s.GameVideo, s.GameFile, &s.GameFile
	}),
)
