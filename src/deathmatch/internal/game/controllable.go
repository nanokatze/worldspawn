package game

import "worldspawn/internal/ecs"

type Controllable interface {
	ControllableUpdateSubtick(w *Scene, id ecs.Entity, cmd TimestampedInputCmd, info *UpdateParams)
	ControllableUpdate(w *Scene, id ecs.Entity, info *UpdateParams)
}
