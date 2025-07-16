package worldspawn

import (
	"time"

	"worldspawn/ecs"
	"worldspawn/geometry-go"
)

type WeaponUpdateInterface interface {
	// TODO: return a more elaborate recoil moment or remove recoil altogether?
	WeaponUpdateSubtick(w *World, id, playerID ecs.ID, now Time, info *UpdateInfo) (recoil geometry.Vec3)
}

type WeaponDeployedInterface interface {
	WeaponDeployed(w *World, id, playerID ecs.ID, now Time, Δt time.Duration)
}

// TODO: we need a more general mechanism
type Viewmodel struct {
	Translation geometry.Vec3
}

// TODO: just remove this component
type WeaponAim struct {
	// TODO: should also include the shooter's entity
	// TODO: remove "Shoot" from the names of these
	ShootPos      geometry.DVec3
	ShootRotation geometry.Rot3
	Buttons       uint64 // TODO: right now this is passing player's buttons, but we also want to plumb stuff like holster away etc
}
