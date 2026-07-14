//go:generate go tool kessoku $GOFILE

package fieldsof_ptr_to_struct

import (
	"github.com/mazrean/kessoku"
)

var StorageSet = kessoku.Set(
	kessoku.Provide(func(s *Storage) (GameImage, *GameImage, GameVideo, *GameVideo) {
		return s.GameImage, &s.GameImage, s.GameVideo, &s.GameVideo
	}),
)
