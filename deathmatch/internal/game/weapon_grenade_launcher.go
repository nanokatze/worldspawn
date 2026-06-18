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

func (weaponState WeaponGrenadeLauncher) CreateProp(world *World, info *UpdateParams) ecs.ID {
	root := world.CreateEntity(info)
	// Ok so what we should do is not parent it to any hands bone. Or maybe we
	// should, but we need a special "weapon" bone I guess and the hands
	// animgraph needs to take into account where the pistol grip and forend etc
	// are. Parenting should be done by the character code.
	// world.ParentBone.Set(root, "hand.R")
	// world.SetTransform(root, gmath.TRS3One[float64]())
	world.SetTransform(root, grenadeLauncherStats.ViewGeometryTRS)
	world.Skeleton.Set(root, "weapons/grenade_launcher/skeletons/Armature")
	world.RenderingGeometry.Set(root, grenadeLauncherStats.RenderingGeometry)
	world.Entity.Set(root, Testburger{
		BaseColor: [4]float32{1, 0.5, 0.5, 1}, // pretend it's a team color
	})

	return root
}

func (weaponState WeaponGrenadeLauncher) WeaponSubstep(
	world *World,
	weapon ecs.ID,
	weaponProps []ecs.ID,
	attacker ecs.ID,
	T gmath.Affine3f64,
	v Velocity,
	buttons WeaponButtons,
	info *UpdateParams) Recoil {
	defer func() { world.Entity.Set(weapon, weaponState) }()

	if buttons&WeaponTrigger != 0 {
		if weaponState.CycleEnds.After(world.Now) {
			return Recoil{}
		}

		if !info.Speculating {
			projectile := world.InstantinatePrefab(grenadeLauncherStats.Projectile, info)
			// TODO: it would be nice if we could specify this bit without assuming ScriptState type
			world.Entity.Set(projectile, GrenadeInFlight{
				LaunchedAt: world.Now,
				Attacker:   attacker,
			})
			world.SetTransform(projectile,
				T.
					Mul(gmath.TRS3f64{
						R: gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{0, 1, 0}),
						S: gmath.Mat3x3UOne[float32](),
					}.Compose()).
					TRS())
			// TODO: consider velocity set on the prefab?
			world.Velocity.Set(projectile,
				Velocity{
					Linear: v.Linear.Add(T.M.Mulv(gmath.Vec3f32{0, grenadeLauncherStats.MuzzleVelocity, 0})),
				})
			world.CollisionLayer.Set(projectile, CollisionLayerProjectiles)
			world.CosmeticOffset.Set(projectile,
				CosmeticOffset{
					Alpha: 2,
					T0:    world.Now,
					// Ugh. TODO: think how we could make this not as gross.
					Offset: T.M.Mulv(gmath.Vec3f32{0.18, 0, -0.2}),
				})
		}

		weaponState.CycleEnds = world.Now.Add(grenadeLauncherStats.CycleDuration)

		// Apply effects to the props; TODO: let's have scripts on the props
		// instead and let props consult the state.
		for _, id := range weaponProps {
			// skelly := world.GetSkeleton(id)
			// world.Pose.Set(id, animgraph.Pose{
			// 	Bones: map[int]gmath.Affine3f32{
			// 		skelly.JointByName("Bolt"): gmath.TRS3f32{
			// 			T: gmath.Vec3f32{0, -0.1 * Rand(world.Now, weaponID, "grenade launcher bolt position").Float32(), 0},
			// 			R: gmath.Rot3One(),
			// 			S: gmath.Mat3x3UOne[float32](),
			// 		}.Compose(),
			// 	},
			// })

			world.SoundEffect.Set(id, SoundEmitter{
				Effect:      "weapons/grenade_launcher/fire.wav",
				Attenuation: 1,
				PlayTime:    world.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
			})
		}

		// TODO: eschew rnd?
		rnd := Rand(world.Now, weapon, T)

		θ := 0.1 * 2 * math.Pi * (rnd.Float64() - 0.5)
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
