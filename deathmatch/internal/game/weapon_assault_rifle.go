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
		) {
			state := weapon.ScriptState().(WeaponAssaultRifle)

			if buttons&WeaponTrigger != 0 && !state.CycleEnds.After(info.Now) {
				// TODO: instead of doing hitscan, spawn a bullet entity, which
				// will be handled by a special system further down the line.
				rayHit := world.TraceRay(
					physics.Ray{
						Origin:    T_attack.T,
						Direction: T_attack.M.Mulv(forward).Normalize(),
						TMax:      1000,
					},
					QueryFilters{
						Entity: func(entity ecs.ID) bool { return entity != attacker.ID() },
					})

				io.Update(attacker, func(info *UpdateParams, mag Entity2, io IO) {
					if mag.Script().Magazine_Pull(info, mag, AmmoBullets, 1, 1, io) <= 0 {
						// play a "click" sound to indicate that we ran out of ammo.
						return
					}

					if !info.Speculating {
						impact := Impact{
							Attacker: attacker,
							Type:     BulletImpact,
							Damage:   7,
							Δv: Velocity{
								Linear: T_attack.M.Mulv(forward).Normalize().Scale(1),
							},
						}

						io.Update(world.GetEntity2(rayHit.Entity), impact.Apply)
					}

					// Apply effects to the props; TODO: let's have scripts on the props
					// instead and let props consult the state.
					for _, id := range weaponProps {
						io.Update(world.GetEntity2(id), func(info *UpdateParams, prop Entity2, io IO) {
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
								Effect:      unique.Make("weapons/grenade_launcher/fire.wav"),
								Attenuation: 1,
								PlayTime:    info.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
							})
						})
					}

					io.Update(weapon, func(info *UpdateParams, weapon Entity2, io IO) {
						state := weapon.ScriptState().(WeaponAssaultRifle)
						defer func() { weapon.SetScriptState(state) }()

						state.CycleEnds = info.Now.Add(time.Second / 8)
					})
				})
			}
		},
	}
}
