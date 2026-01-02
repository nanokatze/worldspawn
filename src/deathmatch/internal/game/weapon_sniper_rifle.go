package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type WeaponSniperRifle struct {
}

func (WeaponSniperRifle) entity() {}

var _ Weapon = WeaponSniperRifle{}

func (weapon WeaponSniperRifle) WeaponCreateGeometry(scene *Scene, parent ecs.ID, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	scene.ParentTo(root, parent)
	scene.SetLocalTRS(root, geometry.DTRS3{
		T: geometry.DVec3{0.15, 0.4, -0.225},
		R: geometry.Rot3One(),
		S: geometry.Vec3Broadcast(1),
	})
	scene.RenderingGeometry.Set(root, "weapons/sniper_rifle/geometries/Sniper_Rifle_001")

	return root
}

func (weapon WeaponSniperRifle) WeaponUpdateSubtick(scene *Scene, weaponID ecs.ID, shootpos geometry.DTRS3, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID) {
	return nil
}
