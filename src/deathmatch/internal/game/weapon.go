package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

// TODO: rename the Weapon interface to just something that can be held

type WeaponButtons uint64

const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

type Weapon interface {
	weapon()

	// TODO: delegate creating the entity to the caller? That way some callers
	// could attach additional stuff if they so desire. But idk.
	WeaponCreateGeometry(scene *Scene, parent ecs.ID, info *UpdateParams) ecs.ID

	// Returns a function that updates the rendering geometry. TODO: we also
	// need to return stuff like recoil and such.
	// TODO: we need to somehow tell the thing to filter the player and possibly
	// other entities
	// TODO: shootpos really should be DVec3 + Rot3 tbh
	WeaponSubstep(scene *Scene, weaponId ecs.ID, shooterID ecs.ID, shootpos geometry.DTRS3, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID)
}
