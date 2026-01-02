package fieldsof_merge

import "github.com/google/wire"

type GameImage interface{}
type GameVideo interface{}
type GameFile interface{}

type Storage struct {
	GameImage GameImage
	GameVideo GameVideo
	GameFile  GameFile
}

var storageSet = wire.NewSet(
	wire.FieldsOf(new(*Storage), "GameImage"),
	wire.FieldsOf(new(*Storage), "GameVideo"),
	wire.FieldsOf(new(*Storage), "GameFile"),
)
