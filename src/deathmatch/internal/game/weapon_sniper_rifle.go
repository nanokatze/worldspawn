package game

import (
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

var sniperRifleStats = struct {
	ViewGeometryTRS geometry.DTRS3 // TODO: this should be killed

	RenderingGeometry string
}{
	ViewGeometryTRS: geometry.DTRS3{
		T: geometry.DVec3{0.15, 0.4, -0.225},
		R: geometry.Rot3One(),
		S: geometry.Vec3Ones(),
	},

	RenderingGeometry: "weapons/sniper_rifle/geometries/Sniper_Rifle_001",
}

type WeaponSniperRifle struct {
	NextAttack time.Duration
}

var _ Weapon = WeaponSniperRifle{}

func (WeaponSniperRifle) entity() {}

func (weapon WeaponSniperRifle) WeaponCreateGeometry(scene *Scene, parent ecs.ID, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	scene.SetParent(root, parent)
	scene.SetLocalTRS(root, sniperRifleStats.ViewGeometryTRS)
	scene.RenderingGeometry.Set(root, sniperRifleStats.RenderingGeometry)

	sound := scene.CreateEntity(info)
	// scene.CreationTime.Set(sound, scene.Now)
	scene.SetParent(sound, root)
	scene.SetLocalTRS(sound, geometry.DTRS3One())
	guh := LoopedSound{
		Sound:       "lamphum.wav", // TODO: don't bother setting it here pls
		Attenuation: 0.1,
	}
	guh.Init()
	scene.SoundEffectState.Set(sound, guh)

	return root
}

func (weapon WeaponSniperRifle) WeaponSubstep(scene *Scene, weaponID ecs.ID, shooterID ecs.ID, shootpos geometry.DTRS3, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID) {
	return nil
}
