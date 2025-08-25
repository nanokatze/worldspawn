package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type WeaponGenericMelee struct {
}

func init() {
	registerEntity[WeaponGenericMelee]()
}

var _ Weapon = WeaponGenericMelee{}

func (weapon WeaponGenericMelee) WeaponUpdateSubtick(w *Scene, weaponID, operatorID ecs.ID, now Time, info *UpdateParams) (recoil geometry.Vec3) {
	w.Entity.Store(weaponID, weapon)
	return geometry.Vec3{}
}
