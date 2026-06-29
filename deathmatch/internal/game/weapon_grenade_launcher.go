package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type Testburger struct {
	BaseColor [4]float32
}

func (Testburger) entity() {}

var grenadeLauncherStats = struct {
	Projectile     string
	MuzzleVelocity float32
	CycleDuration  time.Duration `json:",format:iso8601"`
}{
	Projectile:     "weapons/grenade_launcher_grenade/grenade.json",
	MuzzleVelocity: 30,
	CycleDuration:  600 * time.Millisecond,
}

// TODO: remove the "Weapon" prefix?
type WeaponGrenadeLauncher struct {
	// TODO: make things more interesting
	Chambered bool
	CycleEnds Time
}

func (WeaponGrenadeLauncher) entity() {}

func init() {
	scripts[reflect.TypeFor[WeaponGrenadeLauncher]()] = script{
		WeaponHint: func(world *World, entity ecs.ID) WeaponHint {
			return WeaponHint{
				FirstPersonPropTRS: gmath.TRS3f64{
					T: gmath.Vec3f64{0.18, 0.5, -0.2},
					R: gmath.Rot3One(),
					S: gmath.Mat3x3UOne[float32](),
				},
			}
		},

		WeaponCreateProp: func(world *World, weapon ecs.ID, info *UpdateParams) ecs.ID {
			root := world.CreateEntity(info)
			world.Skeleton.Set(root, "weapons/grenade_launcher/skeletons/Armature")
			world.RenderingGeometry.Set(root, "weapons/grenade_launcher/geometries/Grenade_Launcher")
			world.Entity.Set(root, Testburger{
				BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
			})
			return root
		},

		WeaponThink: func(
			world *World,
			weapon ecs.ID,
			props []ecs.ID,
			attacker ecs.ID,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
			info *UpdateParams,
		) Recoil {
			io := IO{world.Updates, &world.globalUpdates, weapon}

			weaponState, _ := world.GetEntity[WeaponGrenadeLauncher](weapon)

			if weaponState.CycleEnds.After(world.Now) {
				return Recoil{}
			}

			if !weaponState.Chambered {
				magazine := attacker

				io.EnqueueEntityUpdate(magazine, func(world *World, mag ecs.ID, info *UpdateParams) {
					io := IO{world.Updates, &world.globalUpdates, weapon}

					if world.GetScriptFuncs(mag).Magazine_Pull(world, mag, info) {
						io.EnqueueEntityUpdate(weapon,
							func(world *World, weapon ecs.ID, info *UpdateParams) {
								weaponState, _ := world.GetEntity[WeaponGrenadeLauncher](weapon)
								weaponState.Chambered = true
								world.Entity.Set(weapon, weaponState)
							})
					}
				})
				return Recoil{}
			}

			if buttons&WeaponTrigger == 0 {
				return Recoil{}
			}

			if !info.Speculating {
				io.EnqueueGlobalUpdate(func(world *World, info *UpdateParams) {
					// TODO: don't use prefab here tbh
					projectile := world.CreateEntity(info)
					// TODO: it would be nice if we could specify this bit without assuming ScriptState type
					world.Entity.Set(projectile, GrenadeInFlight{
						LaunchedAt: world.Now,
						Attacker:   attacker,
					})
					world.SetTransform(projectile,
						T_attack.
							Mul(gmath.TRS3f64{
								R: gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{0, 1, 0}),
								S: gmath.Mat3x3UOne[float32](),
							}.Compose()).
							TRS())
					world.Velocity.Set(projectile,
						Velocity{
							Linear: v_attack.Linear.Add(T_attack.M.Mulv(forward.Scale(grenadeLauncherStats.MuzzleVelocity))),
						})
					world.CollisionLayer.Set(projectile, CollisionLayerProjectiles)
					world.CosmeticOffset.Set(projectile,
						CosmeticOffset{
							Alpha: 2,
							T0:    world.Now,
							// Ugh. TODO: think how we could make this not as gross.
							Offset: T_attack.M.Mulv(gmath.Vec3f32{0.18, 0, -0.2}),
						})
					world.CollisionGeometry.Set(projectile, "Grenade") // TODO: we should model grenade prop to be something that's kinda 8-gon so that it stops rolling sooner
					world.PhysicsMassOverride.Set(projectile, 0.1)
					world.RenderingGeometry.Set(projectile, "weapons/grenade_launcher_grenade/geometries/Grenade")
				})
			}

			// Apply effects to the props; TODO: let's have scripts on the props
			// instead and let props consult the state.
			for _, id := range props {
				io.EnqueueEntityUpdate(id,
					func(world *World, id ecs.ID, updateParams *UpdateParams) {
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
					})
			}

			io.EnqueueEntityUpdate(weapon,
				func(world *World, weapon ecs.ID, updateParams *UpdateParams) {
					weaponState, _ := world.GetEntity[WeaponGrenadeLauncher](weapon)
					weaponState.Chambered = false
					weaponState.CycleEnds = world.Now.Add(grenadeLauncherStats.CycleDuration)
					world.Entity.Set(weapon, weaponState)
				})

			// TODO: eschew rnd?
			rnd := Rand(world.Now, weapon, T_attack)

			θ := 0.1 * 2 * math.Pi * (rnd.Float64() - 0.5)
			r := 0.02

			return Recoil{
				Recoil: [2]float32{
					float32(math.Sin(θ) * r),
					float32(math.Cos(θ) * r),
				},
			}
		},
	}
}
