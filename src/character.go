package worldspawn

import (
	"time"

	"worldspawn/ecs"
)

// TODO: rename this to something like "accepts input packets"
type Character interface {
	// TODO: this should also take subtick now
	CharacterUpdate(w *World, id ecs.ID, cmd InputCmd2, Δt time.Duration, flags UpdateFlags)
}
