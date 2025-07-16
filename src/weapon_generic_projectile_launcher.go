package worldspawn

import (
	"math"
	"time"

	"worldspawn/ecs"
	"worldspawn/geometry-go"
)

// TODO: I think the way we should do attachments is by spawning two models
// (view and world), with their own set of children entities for attachments.
// Otherwise we run into issues when plumbing stuff to rendering and everything.

type WeaponGenericProjectileLauncher struct {
	DeployAnimation string
	DeployDuration  time.Duration

	// TODO: make a dedicated prefab struct that can be either a filename or *Components
	Projectile     PrefabRef
	MuzzleVelocity float32 // TODO: maybe remove this and let the projectile specify velocity directly. We'd have to transform the spawned prefab in that case
	ShootAnimation string
	ShootSound     string
	CycleDuration  time.Duration `json:",format:units"`

	Armature string // TODO: armatures should be folded directly into animations, probably

	NextAttack Time
}

func init() {
	registerEntity[WeaponGenericProjectileLauncher]()
}

var _ WeaponDeployedInterface = WeaponGenericProjectileLauncher{}

func (weapon WeaponGenericProjectileLauncher) WeaponDeployed(w *World, id, playerID ecs.ID, now Time, Δt time.Duration) {
	weapon.NextAttack = now.Add(weapon.DeployDuration)

	if weapon.DeployAnimation != "" {
		// TODO: this should be two animations: one that moves the root bone (later:
		// the entire entity) and one that moves front cover and tubes

		armature, err := loadArmature(weapon.Armature)
		if err != nil {
			panic(err)
		}

		w.Animation.Store(id, Animation{
			Armature: armature,
			Action:   weapon.DeployAnimation,
			PlayTime: now,
		})
	}

	w.Entity.Store(id, weapon)
}

var _ WeaponUpdateInterface = WeaponGenericProjectileLauncher{}

func (weapon WeaponGenericProjectileLauncher) WeaponUpdateSubtick(w *World, id, playerID ecs.ID, now Time, info *UpdateInfo) (recoil geometry.Vec3) {
	if w.Now < weapon.NextAttack {
		return
	}

	aim, _ := w.WeaponAim.Load(id)
	if aim.Buttons&(1<<ButtonAttack) == 0 {
		return
	}

	// TODO: also spawn a speculative entity on client once we get support for
	// that.
	if info.Speculating {
		rot := aim.ShootRotation

		realPosition := aim.ShootPos.Add(geometry.DVec3FromVec3(rot.Rotate(geometry.Vec3{0.0, 0.5, 0.0})))

		// TODO: cosmeticPosition should be computed from viewmodel
		// TODO: actually, cosmeticShootPos should probably be provided to us, we
		// shouldn't be the ones computing it?
		// TODO: make CosmeticOffset viewmodel-aware.
		cosmeticPosition := aim.ShootPos.Add(geometry.DVec3FromVec3(rot.Rotate(geometry.Vec3{0.15, -0.5, -0.15}).Add(geometry.Vec3{0, 0, -0.1})))

		projectile := w.SpawnPrefab(weapon.Projectile)
		w.TranslationRotation.Store(projectile, TranslationRotation{
			Translation: realPosition,
			Rotation:    rot.Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, math.Pi/2)),
		})
		w.Scale.Store(projectile, geometry.Vec3Broadcast(1))
		// TODO: new idea for cosmetic offset: we could trace a ray like TF2 does
		// and make the decay time be how long it takes to reach the wall!
		w.CosmeticOffset.Store(projectile, CosmeticOffset{
			Offset:    cosmeticPosition.Sub(realPosition).Vec3(),
			StartTime: w.Now,
			EndTime:   w.Now.Add(300 * time.Millisecond),
		})
		w.Velocity.Store(projectile, Velocity{Linear: rot.Rotate(geometry.Vec3{0, weapon.MuzzleVelocity, 0})})
		w.PhysicsLayer.Store(projectile, PhysicsLayerProjectiles)
		w.PhysicsMotionType.Store(projectile, PhysicsMotionDynamic)
		// TODO: which entities to ignore (players might be made out of many
		// entities) and how (some entities are bounding boxes for physics, others
		// can be e.g. hitboxes etc) should be specified through WeaponAim
		w.PhysicsFilter.Store(projectile, []ecs.ID{playerID})
		// w.PhysicsInertiaOverride.Store(projectile, geometry.Mat4x4Diagonal(geometry.Vec4Broadcast(1)))
		w.ResetCosmeticOffsetOnContact.Store(projectile, struct{}{})
		w.SpawnTime.Store(projectile, w.Now)
	}

	w.SoundEffect.Store(id, SoundEffect{
		Effect:   weapon.ShootSound,
		PlayTime: w.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
	})

	if weapon.ShootAnimation != "" {
		armature, err := loadArmature(weapon.Armature)
		if err != nil {
			panic(err)
		}

		w.Animation.Store(id, Animation{
			Armature: armature,
			Action:   weapon.ShootAnimation,
			PlayTime: w.Now,
		})
	}

	weapon.NextAttack = w.Now.Add(weapon.CycleDuration)

	w.Entity.Store(id, weapon)
	return geometry.Vec3{0.1, 0, 0}
}
