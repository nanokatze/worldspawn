package worldspawn

import (
	"time"

	"worldspawn/ecs"
	"worldspawn/geometry-go"
)

type WeaponGenericMelee struct {
}

var _ WeaponUpdateSubtickInterface = WeaponGenericMelee{}

func init() {
	registerEntity[WeaponGenericMelee]()
}

func (weapon WeaponGenericMelee) WeaponUpdateSubtick(w *World, id, playerID ecs.ID, now Time, Δt time.Duration, flags UpdateFlags) (recoil geometry.Vec3) {
	w.Entity.Store(id, weapon)
	return geometry.Vec3{}
}
