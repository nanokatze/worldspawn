package game

import (
	"math"
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type Testburger struct {
	BaseColor [4]float32
}

func (Testburger) entity() {}

// TODO: keep the stats in a json file. We should name the stats file
// explicitly, perhaps in a component.
//
// Or alternatively if we commit to moving entities into scripts, just kill this
// off.
var grenadeLauncherStats = struct {
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

type WeaponGrenadeLauncher struct {
	CycleEnds Time
}

var _ Weapon = WeaponGrenadeLauncher{}

func (WeaponGrenadeLauncher) entity() {}

// TODO: rename to CreateRenderingGeometry
func (weapon WeaponGrenadeLauncher) WeaponCreateGeometry(scene *Scene, parent ecs.ID, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	scene.SetParent(root, parent)
	scene.SetTransform(root, grenadeLauncherStats.ViewGeometryTRS)
	scene.Entity.Set(root, Testburger{
		BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
	})
	scene.RenderingGeometry.Set(root, grenadeLauncherStats.RenderingGeometry)

	return root
}

func (weapon WeaponGrenadeLauncher) WeaponSubstep(scene *Scene, weaponID ecs.ID, shooterID ecs.ID, shootT gmath.Affine3f64, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID) {
	if buttons&WeaponTrigger != 0 {
		if weapon.CycleEnds.After(scene.Now) {
			return nil
		}

		if !info.Speculating {
			projectile := scene.SpawnPrefab(grenadeLauncherStats.Projectile, info)
			// scene.CreationTime.Set(projectile, scene.Now)
			scene.SetTransform(projectile,
				shootT.
					Mul(gmath.TRS3f64{
						R: gmath.Rot3InPlane(gmath.Plane3f32{-1, 0, 0}, math.Pi/2),
						S: gmath.Mat3x3UOne[float32](),
					}.Compose()).
					TRS())
			// TODO: consider velocity set on the prefab?
			scene.Velocity.Set(projectile, Velocity{Linear: shootT.M.Mulv(gmath.Vec3f32{0, grenadeLauncherStats.MuzzleVelocity, 0})})
			scene.CollisionLayer.Set(projectile, PhysicsLayerProjectiles)
			scene.PhysicsFilter.Set(projectile, []ecs.ID{shooterID})
			scene.Timer.Set(projectile, scene.Now.Add(grenadeStats.FuseDuration))
			// TODO: new idea for cosmetic offset: we could trace a ray like TF2 does
			// and make the decay time be how long it takes to reach the wall!
			// scene.CosmeticOffset.Set(projectile, CosmeticOffset{
			// 	Offset:    cosmeticPosition.Sub(realPosition).Vec3(),
			// 	StartTime: scene.Now,
			// 	EndTime:   scene.Now.Add(300 * time.Millisecond),
			// })
			// scene.DeleteCosmeticOffsetOnContact.Set(projectile, struct{}{})
		}

		weapon.CycleEnds = scene.Now.Add(grenadeLauncherStats.CycleDuration)

		scene.Entity.Set(weaponID, weapon)
		return weapon.fired
	}

	return nil
}

func (weapon WeaponGrenadeLauncher) fired(scene *Scene, id ecs.ID) {
	scene.SoundEffect.Set(id, SoundEmitter{
		Effect:      "weapons/grenade_launcher/fire.wav",
		Attenuation: 1,
		PlayTime:    scene.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
	})
}
