package game

import (
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

var sniperRifleStats = struct {
	ViewGeometryTRS gmath.TRS3f64 // TODO: this should be killed

	RenderingGeometry string
}{
	ViewGeometryTRS: gmath.TRS3f64{
		T: gmath.Vec3f64{0.15, 0.4, -0.225},
		R: gmath.Rot3One(),
		S: gmath.Mat3x3UOne[float32](),
	},

	RenderingGeometry: "weapons/sniper_rifle/geometries/Sniper_Rifle_001",
}

type WeaponSniperRifle struct {
	NextAttack time.Duration
}

var _ Weapon = WeaponSniperRifle{}

func (WeaponSniperRifle) entity() {}

func (weapon WeaponSniperRifle) CreateProp(scene *Scene, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	scene.SetTransform(root, sniperRifleStats.ViewGeometryTRS)
	scene.RenderingGeometry.Set(root, sniperRifleStats.RenderingGeometry)

	// sound := scene.CreateEntity(info)
	// // scene.CreationTime.Set(sound, scene.Now)
	// scene.SetParent(sound, root)
	// scene.SetTransform(sound, gmath.TRS3One[float64]())
	// guh := LoopedSound{
	// 	Sound:       "lamphum.wav", // TODO: don't bother setting it here pls
	// 	Attenuation: 0.1,
	// }
	// guh.Init()
	// scene.SoundEffectState.Set(sound, guh)

	return root
}

func (weapon WeaponSniperRifle) WeaponSubstep(
	scene *Scene,
	weaponID ecs.ID,
	propIDs []ecs.ID,
	shooterID ecs.ID,
	T gmath.Affine3f64,
	v Velocity,
	buttons WeaponButtons,
	info *UpdateParams) WeaponStepResult {
	return WeaponStepResult{}
}
