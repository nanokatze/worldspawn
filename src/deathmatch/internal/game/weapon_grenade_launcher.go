package game

import (
	"math"
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type Testburger struct {
	BaseColor [4]float32
}

func (Testburger) entity() {}

// TODO: keep the stats in a json file. We should name the stats file
// explicitly, perhaps in a component.
var grenadeLauncherStats = struct {
	ViewGeometryTRS   geometry.DTRS3 // TODO: this should be killed
	RenderingGeometry string

	Projectile     PrefabRef
	MuzzleVelocity float32
	CycleDuration  time.Duration `json:",format:iso8601"`
}{
	ViewGeometryTRS: geometry.DTRS3{
		T: geometry.DVec3{0.2, 0.4, -0.275},
		R: geometry.Rot3One(),
		S: geometry.Vec3Broadcast(1),
	},
	RenderingGeometry: "weapons/grenade_launcher/geometries/Grenade_Launcher",

	Projectile:     PrefabRef{Filename: "weapons/grenade_launcher_grenade/grenade.json"},
	MuzzleVelocity: 30,
	CycleDuration:  600 * time.Millisecond,
}

type WeaponGrenadeLauncher struct {
	// TODO: rename
	NextAttack Time
}

func (WeaponGrenadeLauncher) entity() {}

var _ Weapon = WeaponGrenadeLauncher{}

func (WeaponGrenadeLauncher) weapon() {}

// TODO: rename to CreateRenderingGeometry
func (weapon WeaponGrenadeLauncher) WeaponCreateGeometry(scene *Scene, parent ecs.ID, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	scene.SetParent(root, parent)
	scene.SetLocalTRS(root, grenadeLauncherStats.ViewGeometryTRS)
	scene.Entity.Set(root, Testburger{
		BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
	})
	scene.RenderingGeometry.Set(root, grenadeLauncherStats.RenderingGeometry)

	return root
}

func (weapon WeaponGrenadeLauncher) WeaponSubstep(scene *Scene, weaponID ecs.ID, shooterID ecs.ID, shootpos geometry.DTRS3, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID) {
	if buttons&WeaponTrigger != 0 {
		if !weapon.NextAttack.After(scene.Now) {
			if !info.Speculating {
				projectile := scene.SpawnPrefab(grenadeLauncherStats.Projectile, 0, info)
				scene.CreationTime.Set(projectile, scene.Now)
				scene.SetGlobalTRS(projectile, shootpos.Mul(geometry.DTRS3{
					R: geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, math.Pi/2),
					S: geometry.Vec3Broadcast(1),
				}))
				// TODO: consider velocity set on the prefab?
				scene.Velocity.Set(projectile, Velocity{Linear: shootpos.R.Rotate(geometry.Vec3{0, grenadeLauncherStats.MuzzleVelocity, 0})})
				scene.CollisionLayer.Set(projectile, PhysicsLayerProjectiles)
				scene.PhysicsFilter.Set(projectile, []ecs.ID{shooterID})
				// TODO: new idea for cosmetic offset: we could trace a ray like TF2 does
				// and make the decay time be how long it takes to reach the wall!
				// scene.CosmeticOffset.Set(projectile, CosmeticOffset{
				// 	Offset:    cosmeticPosition.Sub(realPosition).Vec3(),
				// 	StartTime: scene.Now,
				// 	EndTime:   scene.Now.Add(300 * time.Millisecond),
				// })
				// scene.DeleteCosmeticOffsetOnContact.Set(projectile, struct{}{})
			}

			weapon.NextAttack = scene.Now.Add(grenadeLauncherStats.CycleDuration)

			scene.Entity.Set(weaponID, weapon)
			return weapon.fired
		}
	}

	return nil
}

// TODO: we could also make it a method on the proj launcher tbh?
func (weapon WeaponGrenadeLauncher) fired(scene *Scene, id ecs.ID) {
	scene.SoundEffect.Set(id, SoundEmitter{
		Effect:      "weapons/grenade_launcher/fire.wav",
		Attenuation: 1,
		PlayTime:    scene.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
	})
}
