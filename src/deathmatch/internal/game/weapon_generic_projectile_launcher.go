package game

import (
	"math"
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type WeaponGenericProjectileLauncher struct {
	Projectile     PrefabRef
	MuzzleVelocity float32
	CycleDuration  time.Duration `json:",format:units"`

	// TODO: rename
	NextAttack Time
}

type Testburger struct {
	BaseColor [4]float32
}

var _ Weapon = WeaponGenericProjectileLauncher{}

// TODO: rename to something else like CreateVisual or CreateRenderingGeometry
func (weapon WeaponGenericProjectileLauncher) WeaponCreateGeometry(scene *Scene, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	scene.TranslationRotation.Set(root, TranslationRotation{
		Translation: geometry.DVec3{0.2, 0.4, -0.275},
		Rotation:    geometry.Rot3One(),
	})
	scene.RenderingGeometry.Set(root, PackGeometry(Geometry{Kind: GeometryFileBacked, Filename: "weapons/grenade_launcher/geometries/Grenade_Launcher"}))
	scene.Entity.Set(root, Testburger{
		BaseColor: [4]float32{1, 0, 0, 1}, // pretend it's a team color
	})

	return root
}

func (weapon WeaponGenericProjectileLauncher) WeaponUpdateSubtick(scene *Scene, weaponID ecs.ID, shootpos TranslationRotation, buttons WeaponButtons, info *UpdateParams) func(*Scene, ecs.ID) {
	if buttons&WeaponTrigger != 0 {
		if !weapon.NextAttack.After(scene.Now) {
			// TODO: spawn entity here

			if !info.Speculating {
				projectile := scene.SpawnPrefab(weapon.Projectile, info)
				scene.CreationTime.Set(projectile, scene.Now)
				scene.TranslationRotation.Set(projectile, TranslationRotation{
					Translation: shootpos.Translation,
					Rotation:    shootpos.Rotation.Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, math.Pi/2)),
				})
				// TODO: consider velocity set on the prefab?
				scene.Velocity.Set(projectile, Velocity{Linear: shootpos.Rotation.Rotate(geometry.Vec3{0, weapon.MuzzleVelocity, 0})})
				// TODO: new idea for cosmetic offset: we could trace a ray like TF2 does
				// and make the decay time be how long it takes to reach the wall!
				// scene.CosmeticOffset.Set(projectile, CosmeticOffset{
				// 	Offset:    cosmeticPosition.Sub(realPosition).Vec3(),
				// 	StartTime: scene.Now,
				// 	EndTime:   scene.Now.Add(300 * time.Millisecond),
				// })
				// scene.DeleteCosmeticOffsetOnContact.Set(projectile, struct{}{})
				scene.CollisionLayer.Set(projectile, PhysicsLayerProjectiles)
				// TODO: which entities to ignore (players might be made out of many
				// entities) and how (some entities are bounding boxes for physics, others
				// can be e.g. hitboxes etc) should be specified through WeaponAim
				// scene.PhysicsFilter.Set(projectile, []ecs.Entity{playerID})
				// scene.PhysicsInertiaOverride.Store(projectile, geometry.Mat4x4Diagonal(geometry.Vec4Broadcast(1)))
			}

			weapon.NextAttack = scene.Now.Add(weapon.CycleDuration)

			scene.Entity.Set(weaponID, weapon)
			return weapon.fired
		}
	}

	return nil
}

// TODO: we could also make it a method on the proj launcher tbh?
func (weapon WeaponGenericProjectileLauncher) fired(scene *Scene, id ecs.ID) {
	scene.SoundEffect.Set(id, SoundEmitter{
		Effect:   "weapons/grenade_launcher/fire.wav",
		PlayTime: scene.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
	})
}
