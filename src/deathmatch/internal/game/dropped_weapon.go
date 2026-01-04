package game

import "worldspawn/internal/ecs"

type DroppedWeapon struct {
	Weapon ecs.ID
}

func (DroppedWeapon) entity() {}
