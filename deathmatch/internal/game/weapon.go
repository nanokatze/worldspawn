package game

import (
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type WeaponButtons uint64

const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

// TODO: rename this to literally anything but
type Recoil struct {
	Recoil [2]float32
}

type Weapon interface {
	Entity

	CreateProp(world *World, info *UpdateParams) ecs.ID

	// Returns a function that updates the rendering geometry.
	// TODO: we also need to return stuff like recoil and such.
	// TODO: we need to somehow tell the thing to filter the shooter and
	// possibly other entities
	WeaponSubstep(
		world *World,
		weaponID ecs.ID,
		propIDs []ecs.ID,
		shooterID ecs.ID,
		T gmath.Affine3f64,
		v Velocity,
		buttons WeaponButtons,
		info *UpdateParams,
	) Recoil
}
