package worldspawn

import (
	"worldspawn/ecs"
	"worldspawn/geometry-go"
)

type WeaponGenericMelee struct {
}

func init() {
	registerEntity[WeaponGenericMelee]()
}

var _ WeaponUpdateInterface = WeaponGenericMelee{}

func (weapon WeaponGenericMelee) WeaponUpdateSubtick(w *World, id, playerID ecs.ID, now Time, info *UpdateInfo) (recoil geometry.Vec3) {
	w.Entity.Store(id, weapon)
	return geometry.Vec3{}
}
