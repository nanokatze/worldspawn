package game

import (
	"reflect"
	"time"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

type WeaponAssaultRifle struct {
	Chambered bool
	CycleEnds Time
}

func init() {
	Scripts[reflect.TypeFor[WeaponAssaultRifle]()] = script{
		Weapon_Hint: func(info *UpdateParams, world *World, entity ecs.ID) WeaponHint {
			return WeaponHint{
				FirstPersonPropTransform: gmath.TRS3f64{
					T: gmath.Vec3f64{0.18, 0.5, -0.2},
					R: gmath.Rot3One(),
					S: gmath.Mat3x3UOne[float32](),
				},
			}
		},

		Weapon_CreateProp: func(info *UpdateParams, world *World, weapon ecs.ID) Entity2 {
			root := world.CreateEntity(info)
			root.SetScriptState(Testburger{
				BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
			})
			root.SetRenderingGeometry(unique.Make("weapons/grenade_launcher/geometries/Grenade_Launcher"))
			return root
		},

		Weapon_Think: func(
			info *UpdateParams,
			world *World,
			weapon Entity2,
			weaponProps []ecs.ID,
			attacker Entity2,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
			io IO,
		) Recoil {
			state := weapon.ScriptState().(WeaponAssaultRifle)

			if state.CycleEnds.After(info.Now) {
				return Recoil{}
			}

			// TODO: move this state transition into the normal Think. We'll
			// need to be able to hold onto the player to pull ammo off them.
			if !state.Chambered {
				io.EnqueueEntityUpdate(attacker,
					func(info *UpdateParams, mag Entity2, io IO) {
						if mag.Script().Magazine_Pull(info, mag, 1, io) {
							io.EnqueueEntityUpdate(weapon,
								func(info *UpdateParams, weapon Entity2, io IO) {
									state := weapon.ScriptState().(WeaponAssaultRifle)
									defer func() { weapon.SetScriptState(state) }()

									state.Chambered = true
								})
						}
					})
				return Recoil{}
			}

			if buttons&WeaponTrigger == 0 {
				return Recoil{}
			}

			if !info.Speculating {
				rayHit := world.TraceRay(
					physics.Ray{
						Origin:    T_attack.T,
						Direction: T_attack.M.Mulv(forward).Normalize(),
						TMax:      1000,
					},
					QueryFilters{
						Entity: func(entity ecs.ID) bool { return entity != attacker.ID() },
					})

				if rayHit.Entity != ecs.NullID {
					// TODO: fill-in Δv
					io.EnqueueEntityUpdate(world.GetEntity2(rayHit.Entity), Impact{Attacker: attacker.ID(), Type: BulletImpact, Damage: 7}.Apply)
				}

				/*
					io.EnqueueGlobalUpdate(func(info *UpdateParams, world *World) {
						// TODO: don't use prefab here tbh
						projectile := world.CreateEntity(info)
						// TODO: it would be nice if we could specify this bit without assuming ScriptState type
						projectile.SetScriptState(GrenadeInFlight{
							LaunchedAt: world.Now,
							Attacker:   attacker,
						})
						projectile.SetTransform(
							T_attack.
								Mul(gmath.TRS3f64{
									R: gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{0, 1, 0}),
									S: gmath.Mat3x3UOne[float32](),
								}.Compose()).
								TRS())
						projectile.SetVelocity(Velocity{
							Linear: v_attack.Linear.Add(T_attack.M.Mulv(forward.Scale(grenadeLauncherStats.MuzzleVelocity))),
						})
						projectile.SetCollisionLayer(CollisionLayerProjectiles)
						// TODO: we should model grenade prop to be something that's kinda 8-gon so that it stops rolling sooner
						projectile.SetCollisionGeometry(unique.Make("Grenade"))
						projectile.SetPhysicsMassOverride(0.1)
						projectile.SetCosmeticOffset(CosmeticOffset{
							Alpha: 2,
							T0:    world.Now,
							// Ugh. TODO: think how we could make this not as gross.
							Offset: T_attack.M.Mulv(gmath.Vec3f32{0.18, 0, -0.2}),
						})
						projectile.SetRenderingGeometry(unique.Make("weapons/grenade_launcher_grenade/geometries/Grenade"))
					})
				*/
			}

			// Apply effects to the props; TODO: let's have scripts on the props
			// instead and let props consult the state.
			for _, id := range weaponProps {
				io.EnqueueEntityUpdate(world.GetEntity2(id),
					func(info *UpdateParams, prop Entity2, io IO) {
						// skelly := world.GetSkeleton(prop)
						// world.Pose.Set(prop, animgraph.Pose{
						// 	Bones: map[int]gmath.Affine3f32{
						// 		skelly.JointByName("Bolt"): gmath.TRS3f32{
						// 			T: gmath.Vec3f32{0, -0.1 * Rand(world.Now, weaponID, "grenade launcher bolt position").Float32(), 0},
						// 			R: gmath.Rot3One(),
						// 			S: gmath.Mat3x3UOne[float32](),
						// 		}.Compose(),
						// 	},
						// })

						prop.SetSoundEffect(SoundEmitter{
							Effect:      "weapons/grenade_launcher/fire.wav",
							Attenuation: 1,
							PlayTime:    info.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
						})
					})
			}

			io.EnqueueEntityUpdate(weapon,
				func(info *UpdateParams, weapon Entity2, io IO) {
					state := weapon.ScriptState().(WeaponAssaultRifle)
					defer func() { weapon.SetScriptState(state) }()

					state.Chambered = false
					state.CycleEnds = info.Now.Add(time.Second / 10)
				})

			return Recoil{}
		},
	}
}
