package game

import (
	"math"
	"reflect"
	"time"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

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
	CycleEnds Time
}

type Testburger struct {
	BaseColor [4]float32
}

func init() {
	Scripts[reflect.TypeFor[Testburger]()] = script{}

	Scripts[reflect.TypeFor[WeaponGrenadeLauncher]()] = script{
		Weapon_Hint: func(info *UpdateParams, world *World, entity ecs.ID) WeaponHint {
			return WeaponHint{
				FirstPersonPropTransform: gmath.TRS3f64{
					T: gmath.Vec3f64{0.18, 0.5, -0.2},
					R: gmath.Rot3One(),
					S: gmath.Mat3x3UOne[float32](),
				},
			}
		},

		Weapon_CreateProp: func(stx ScriptContext, weapon Entity2, f func(stx ScriptContext, entity Entity2)) {
			stx.Create(func(stx ScriptContext, prop Entity2) {
				prop.SetScriptState(Testburger{
					BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
				})
				prop.SetSkeleton(unique.Make("weapons/grenade_launcher/skeletons/Armature"))
				prop.SetRenderingGeometry(unique.Make("weapons/grenade_launcher/geometries/Grenade_Launcher"))
				f(stx, prop)
			})
		},

		Weapon_Think: func(
			stx ScriptContext,
			world *World,
			weapon Entity2,
			weaponProps []Entity2,
			attacker Entity2,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
		) {
			state := weapon.ScriptState().(WeaponGrenadeLauncher)

			if buttons&WeaponTrigger != 0 && !state.CycleEnds.After(stx.Now) {
				stx.Update(attacker, func(stx ScriptContext, mag Entity2) {
					if mag.Script().Magazine_Pull(stx, mag, AmmoGrenades, 1, 1) <= 0 {
						return
					}

					if !stx.Speculating {
						stx.Create(func(stx ScriptContext, projectile Entity2) {
							// TODO: don't use prefab here tbh
							// TODO: it would be nice if we could specify this bit without assuming ScriptState type
							projectile.SetScriptState(GrenadeInFlight{
								Attacker:   attacker.ID(),
								LaunchedAt: stx.Now,
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
								T0:    stx.Now,
								// Ugh. TODO: think how we could make this not as gross.
								Offset: T_attack.M.Mulv(gmath.Vec3f32{0.18, 0, -0.2}),
							})
							projectile.SetRenderingGeometry(unique.Make("weapons/grenade_launcher_grenade/geometries/Grenade"))
						})
					}

					stx.Update(weapon, func(stx ScriptContext, weapon Entity2) {
						state := weapon.ScriptState().(WeaponGrenadeLauncher)
						defer func() { weapon.SetScriptState(state) }()

						state.CycleEnds = stx.Now.Add(grenadeLauncherStats.CycleDuration)
					})

					// Apply effects to the props; TODO: let's have scripts on the props
					// instead and let props consult the state.
					for _, prop := range weaponProps {
						stx.Update(prop, func(stx ScriptContext, prop Entity2) {
							// skelly := world.GetSkeleton(prop)
							// world.Pose.Set(prop, animgraph.Pose{
							// 	Bones: map[int]gmath.Affine3f32{
							// 		skelly.JointByName("Bolt"): gmath.TRS3f32{
							// 			T: gmath.Vec3f32{0, -0.1 * Rand(info.Now, weaponID, "grenade launcher bolt position").Float32(), 0},
							// 			R: gmath.Rot3One(),
							// 			S: gmath.Mat3x3UOne[float32](),
							// 		}.Compose(),
							// 	},
							// })

							prop.SetSoundEffect(SoundEmitter{
								Effect:      unique.Make("weapons/grenade_launcher/fire.wav"),
								Attenuation: 1,
								PlayTime:    stx.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
							})
						})
					}

					// TODO: eschew rnd?
					rnd := Rand(stx.Now, weapon.ID(), T_attack)

					θ := 0.1 * 2 * math.Pi * (rnd.Float64() - 0.5)
					r := 0.02

					_ = [2]float32{
						float32(math.Sin(θ) * r),
						float32(math.Cos(θ) * r),
					}
				})
			}
		},
	}
}
