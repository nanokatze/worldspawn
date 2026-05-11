package game

import (
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

// TODO: rename the Weapon interface to just something that can be held?

type WeaponButtons uint64

const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

type WeaponStepResult struct {
	Recoil [2]float32
}

type Weapon interface {
	Entity

	CreateProp(scene *Scene, info *UpdateParams) ecs.ID

	// Returns a function that updates the rendering geometry.
	// TODO: we also need to return stuff like recoil and such.
	// TODO: we need to somehow tell the thing to filter the shooter and
	// possibly other entities
	WeaponSubstep(
		scene *Scene,
		weaponID ecs.ID,
		propIDs []ecs.ID,
		shooterID ecs.ID,
		T gmath.Affine3f64,
		v Velocity,
		buttons WeaponButtons,
		info *UpdateParams,
	) WeaponStepResult
}
