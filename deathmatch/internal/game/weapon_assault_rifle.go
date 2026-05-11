package game

import (
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

var assaultRifleStats = struct {
	ViewGeometryTRS   gmath.TRS3f64 // TODO: this should be killed
	RenderingGeometry string

	Projectile     string
	MuzzleVelocity float32
	CycleDuration  time.Duration `json:",format:iso8601"`
}{
	ViewGeometryTRS: gmath.TRS3f64{
		T: gmath.Vec3f64{0.18, 0.5, -0.2},
		R: gmath.Rot3One(),
		S: gmath.Mat3x3UOne[float32](),
	},
	RenderingGeometry: "weapons/grenade_launcher/geometries/Grenade_Launcher",

	Projectile:     "weapons/grenade_launcher_grenade/grenade.json",
	MuzzleVelocity: 30,
	CycleDuration:  600 * time.Millisecond,
}

// TODO: remove the "Weapon" prefix?
type WeaponAssaultRifle struct {
	CycleEnds Time
}

var _ Weapon = WeaponAssaultRifle{}

func (WeaponAssaultRifle) entity() {}

// TODO: rename to CreateProp?
func (weapon WeaponAssaultRifle) CreateProp(scene *Scene, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	// Ok so what we should do is not parent it to any hands bone. Or maybe we
	// should, but we need a special "weapon" bone I guess and the hands
	// animgraph needs to take into account where the pistol grip and forend etc
	// are. Parenting should be done by the character code.
	// scene.ParentBone.Set(root, "hand.R")
	// scene.SetTransform(root, gmath.TRS3One[float64]())
	scene.SetTransform(root, assaultRifleStats.ViewGeometryTRS)
	scene.Skeleton.Set(root, "weapons/grenade_launcher/skeletons/Armature")
	scene.RenderingGeometry.Set(root, assaultRifleStats.RenderingGeometry)
	scene.Entity.Set(root, Testburger{
		BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
	})

	return root
}

func (weapon WeaponAssaultRifle) WeaponSubstep(
	scene *Scene,
	weaponID ecs.ID,
	propIDs []ecs.ID,
	shooterID ecs.ID,
	T gmath.Affine3f64,
	v Velocity,
	buttons WeaponButtons,
	info *UpdateParams) WeaponStepResult {
	defer func() { scene.Entity.Set(weaponID, weapon) }()

	return WeaponStepResult{}
}
