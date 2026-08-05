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

		Weapon_CreateProp: func(stx ScriptContext, weapon Entity, f func(stx ScriptContext, entity Entity)) {
			stx.Create(func(stx ScriptContext, prop Entity) {
				prop.SetScriptState(Testburger{
					BaseColor: [4]float32{1, 0.1, 0.1, 1},
				})
				prop.SetSkeleton(unique.Make("weapons/grenade_launcher/skeletons/Armature"))
				prop.SetRenderingGeometry(unique.Make("weapons/grenade_launcher/geometries/Grenade_Launcher"))
				f(stx, prop)
			})
		},

		Weapon_Think: func(
			stx ScriptContext,
			world *World,
			weapon Entity,
			weaponProps []Entity,
			attacker Entity,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
		) {
			state := weapon.ScriptState().(WeaponAssaultRifle)

			if buttons&WeaponTrigger != 0 && !state.CycleEnds.After(stx.Now) {
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

				stx.Update(attacker, func(stx ScriptContext, mag Entity) {
					if mag.Script().Magazine_Pull(stx, mag, AmmoBullets, 1, 1) <= 0 {
						// play a "click" sound to indicate that we ran out of ammo.
						return
					}

					if !stx.Speculating {
						if rayHit.Entity.IsValid() {
							impact := Impact{
								Attacker: attacker,
								Type:     BulletImpact,
								Damage:   7,
								Δv: Velocity{
									Linear: T_attack.M.Mulv(forward).Normalize().Scale(1),
								},
							}

							stx.Update(rayHit.Entity, impact.Apply)
						}
					}

					// Apply effects to the props; TODO: let's have scripts on the props
					// instead and let props consult the state.
					for _, prop := range weaponProps {
						stx.Update(prop, func(stx ScriptContext, prop Entity2) {
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
								Effect:      unique.Make("weapons/grenade_launcher/sounds/Fire"),
								Attenuation: 1,
								PlayTime:    stx.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
							})
						})
					}

					stx.Update(weapon, func(stx ScriptContext, weapon Entity) {
						state := weapon.ScriptState().(WeaponAssaultRifle)
						defer func() { weapon.SetScriptState(state) }()

						state.CycleEnds = stx.Now.Add(time.Second / 8)
					})
				})
			}
		},
	}
}
