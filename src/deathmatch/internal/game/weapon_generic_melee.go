//go:build ignore

package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type WeaponGenericMelee struct {
}

var _ Weapon = WeaponGenericMelee{}

func (weapon WeaponGenericMelee) WeaponUpdateSubtick(w *Scene, weaponID, operatorID ecs.Entity, now Time, info *UpdateParams) (recoil geometry.Vec3) {
	w.Entity.Set(weaponID, weapon)
	return geometry.Vec3{}
}
