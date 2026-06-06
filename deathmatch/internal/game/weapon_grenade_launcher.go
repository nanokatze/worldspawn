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

// TODO: remove the "Weapon" prefix?
type WeaponGrenadeLauncher struct {
	CycleEnds Time
}

var _ Weapon = WeaponGrenadeLauncher{}

func (WeaponGrenadeLauncher) entity() {}

func (weapon WeaponGrenadeLauncher) CreateProp(scene *Scene, info *UpdateParams) ecs.ID {
	root := scene.CreateEntity(info)
	// Ok so what we should do is not parent it to any hands bone. Or maybe we
	// should, but we need a special "weapon" bone I guess and the hands
	// animgraph needs to take into account where the pistol grip and forend etc
	// are. Parenting should be done by the character code.
	// scene.ParentBone.Set(root, "hand.R")
	// scene.SetTransform(root, gmath.TRS3One[float64]())
	scene.SetTransform(root, grenadeLauncherStats.ViewGeometryTRS)
	scene.Skeleton.Set(root, "weapons/grenade_launcher/skeletons/Armature")
	scene.RenderingGeometry.Set(root, grenadeLauncherStats.RenderingGeometry)
	scene.Entity.Set(root, Testburger{
		BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
	})

	return root
}

func (weapon WeaponGrenadeLauncher) WeaponSubstep(
	scene *Scene,
	weaponID ecs.ID,
	propIDs []ecs.ID,
	shooterID ecs.ID,
	T gmath.Affine3f64,
	v Velocity,
	buttons WeaponButtons,
	info *UpdateParams) Recoil {
	defer func() { scene.Entity.Set(weaponID, weapon) }()

	if buttons&WeaponTrigger != 0 {
		if weapon.CycleEnds.After(scene.Now) {
			return Recoil{}
		}

		if !info.Speculating {
			projectile := scene.SpawnPrefab(grenadeLauncherStats.Projectile, info)

			scene.SetTransform(projectile,
				T.
					Mul(gmath.TRS3f64{
						R: gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{0, 1, 0}),
						S: gmath.Mat3x3UOne[float32](),
					}.Compose()).
					TRS())
			// TODO: consider velocity set on the prefab?
			scene.Velocity.Set(projectile,
				Velocity{
					Linear: v.Linear.Add(T.M.Mulv(gmath.Vec3f32{0, grenadeLauncherStats.MuzzleVelocity, 0})),
				})
			scene.CollisionLayer.Set(projectile, CollisionLayerProjectiles)
			scene.PhysicsFilter.Set(projectile, []ecs.ID{shooterID})
			scene.NextThink.Set(projectile, scene.Now.Add(1400*time.Millisecond)) // TODO: this should be put into grenade somehow
			scene.CosmeticOffset.Set(projectile,
				CosmeticOffset{
					Alpha: 2,
					T0:    scene.Now,
					// Ugh. TODO: think how we could make this not as gross.
					Offset: T.M.Mulv(gmath.Vec3f32{0.18, 0, -0.2}),
				})
			scene.DeleteCosmeticOffsetOnContact.Set(projectile, struct{}{})
		}

		weapon.CycleEnds = scene.Now.Add(grenadeLauncherStats.CycleDuration)

		// Apply effects to the props; TODO: let's have scripts on the props
		// instead and let props consult the state.
		for _, id := range propIDs {
			// skelly := scene.GetSkeleton(id)
			// scene.Pose.Set(id, animgraph.Pose{
			// 	Bones: map[int]gmath.Affine3f32{
			// 		skelly.JointByName("Bolt"): gmath.TRS3f32{
			// 			T: gmath.Vec3f32{0, -0.1 * Rand(scene.Now, weaponID, "grenade launcher bolt position").Float32(), 0},
			// 			R: gmath.Rot3One(),
			// 			S: gmath.Mat3x3UOne[float32](),
			// 		}.Compose(),
			// 	},
			// })

			scene.SoundEffect.Set(id, SoundEmitter{
				Effect:      "weapons/grenade_launcher/fire.wav",
				Attenuation: 1,
				PlayTime:    scene.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
			})
		}

		// TODO: eschew rng?
		rng := Rand(scene.Now, weaponID, T)

		θ := 0.1 * 2 * math.Pi * (rng.Float64() - 0.5)
		r := 0.02

		return Recoil{
			Recoil: [2]float32{
				float32(math.Sin(θ) * r),
				float32(math.Cos(θ) * r),
			},
		}
	}

	return Recoil{}
}
