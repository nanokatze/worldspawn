package game

import (
	"worldspawn/internal/ecs"
)

type WeaponButtons uint64

const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

type Weapon interface {
	// Only call this on the server
	CreateGeometry(scene *Scene, info *UpdateParams) ecs.Entity

	// Returns a function that updates the visual. TODO: we also need to return
	// stuff like recoil and such.
	UpdateSubtick(scene *Scene, weaponId ecs.Entity, buttons WeaponButtons, info *UpdateParams) func(ecs.Entity)
}
