package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type WeaponButtons uint64

const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

type Weapon interface {
	WeaponCreateGeometry(scene *Scene, info *UpdateParams) ecs.ID

	// Returns a function that updates the visual. TODO: we also need to return
	// stuff like recoil and such.
	// TODO: we need to somehow tell the thing to filter the player and possibly
	// other entities
	WeaponUpdateSubtick(scene *Scene, weaponId ecs.ID, shootpos geometry.DTRS3, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID)
}
