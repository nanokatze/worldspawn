package game

import (
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

// TODO: swap weaponID and operatorID places?

type Weapon interface {
	// TODO: return a more elaborate recoil moment or remove recoil altogether?
	WeaponUpdateSubtick(w *Scene, weaponID, operatorID ecs.ID, now Time, info *UpdateParams) (recoil geometry.Vec3)
}

// TODO: rename this
type WeaponDeployedInterface interface {
	WeaponDeployed(w *Scene, weaponID, operatorID ecs.ID, now Time, Δt time.Duration)
}

// TODO: just remove this component
type WeaponAim struct {
	// TODO: should also include the shooter's entity
	// TODO: remove "Shoot" from the names of these
	ShootPos      geometry.DVec3
	ShootRotation geometry.Rot3
	Buttons       uint64 // TODO: right now this is passing player's buttons, but we also want to plumb stuff like holster away etc
}
