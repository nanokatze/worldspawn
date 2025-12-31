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
	WeaponCreateGeometry(scene *Scene, info *UpdateParams) ecs.Entity

	// Returns a function that updates the visual. TODO: we also need to return
	// stuff like recoil and such.
	WeaponUpdateSubtick(scene *Scene, weaponId ecs.Entity, shootpos TranslationRotation, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.Entity)
}
