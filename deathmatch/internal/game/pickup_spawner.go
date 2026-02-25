package game

import (
	"worldspawn/internal/ecs"
)

type PickupSpawner struct {
}

func (PickupSpawner) entity() {}

// TODO: move handling of this into Character?
func (pickupSpawner PickupSpawner) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	// TODO: we should just spawn DroppedWeapon here?
}
