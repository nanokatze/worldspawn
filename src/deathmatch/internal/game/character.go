package game

import (
	"worldspawn/internal/ecs"
)

// TODO: rename this to something like "accepts input packets"
type Character interface {
	// TODO: this should also take subtick now
	CharacterUpdate(w *Scene, id ecs.Entity, cmd TimestampedInputCmd, info *UpdateParams)
}
