package fieldsof_ptr_to_struct

import "github.com/google/wire"

type GameImage struct{}
type GameVideo struct{}

type Storage struct {
	GameImage GameImage
	GameVideo GameVideo
}

// new(*Storage) triggers isPtrToStruct=true in wire, which provides both
// GameImage and *GameImage (and GameVideo/*GameVideo) as outputs.
var StorageSet = wire.NewSet(wire.FieldsOf(new(*Storage), "GameImage", "GameVideo"))
