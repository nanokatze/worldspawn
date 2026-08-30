package game

import (
	"math"
	"reflect"
	"time"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
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
	Animation unique.Handle[string]

	PlayTime      Time
	PlaybackSpeed float32 // 0 is remapped to 1

	BaseColor [4]float32
}

func init() {
	Scripts[reflect.TypeFor[Testburger]()] = script{
		Think: func(stx ScriptContext, entity Entity, world *World) {
			state := entity.ScriptState().(Testburger)

			if state.Animation != (unique.Handle[string]{}) {
				stx.Update(entity, func(stx ScriptContext, entity Entity) {
					anim := animationCache.Get(state.Animation)

					sk := skeletonCache.Get(entity.Skeleton())

					playbackSpeed := state.PlaybackSpeed
					if playbackSpeed == 0 {
						playbackSpeed = 1
					}

					v := make([]float32, len(anim.Channels()))
					animation.SampleTime(anim, time.Duration(1e9*(durationToFloatSeconds(stx.Now.Sub(state.PlayTime))*float64(playbackSpeed))), v)

					localTransforms := make([]gmath.Affine3f32, sk.NumJoints())

					poseMapperCache.Get(poseMapperKey{anim, sk})(v, localTransforms)

					pose := make(skeleton.Pose, sk.NumJoints())
					sk.ForwardKinematics(localTransforms, pose)

					entity.SetPose(pose)
				})
			}
		},
	}

	Scripts[reflect.TypeFor[WeaponGrenadeLauncher]()] = script{
		Weapon_Hint: func(info *UpdateParams, entity Entity) WeaponHint {
			return WeaponHint{
				DrawDurationMultiplier: 1,
				HideDurationMultiplier: 1,

				FirstPersonPropTransform: gmath.Affine3TRSf64{
					T: gmath.Vec3f64{0.18, 0.5, -0.2},
					R: gmath.Rot3One(),
					S: gmath.Mat3x3UOne[float32](),
				},
			}
		},

		Weapon_CreateProp: func(stx ScriptContext, weapon Entity, f func(stx ScriptContext, entity Entity)) {
			stx.Create(func(stx ScriptContext, prop Entity) {
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
			weapon Entity,
			weaponProps []Entity,
			world *World,
			attacker Entity,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
			recoil func(stx ScriptContext, recoil [2]float32),
		) {
			state := weapon.ScriptState().(WeaponGrenadeLauncher)

			if buttons&WeaponTrigger != 0 && state.CycleEnds.Compare(stx.Now) <= 0 {
				stx.Update(attacker, func(stx ScriptContext, mag Entity) {
					if mag.Script().Magazine_Pull(stx, mag, AmmoGrenades, 1, 1) <= 0 {
						return
					}

					if !stx.Speculating {
						stx.Create(func(stx ScriptContext, projectile Entity) {
							projectile.SetScriptState(GrenadeInFlight{
								Attacker:   attacker.ID(),
								LaunchedAt: stx.Now,
							})
							projectile.SetTransform(gmath.Affine3DecomposeTRS(
								T_attack.
									Mul(gmath.Affine3TRSf64{
										R: gmath.Rot3AToB(up, forward),
										S: gmath.Mat3x3UOne[float32](),
									}.Affine())))
							projectile.SetVelocity(Velocity{
								Linear: v_attack.Linear.Add(gmath.Matvec3(T_attack.M, forward.Scale(grenadeLauncherStats.MuzzleVelocity))),
							})
							projectile.SetCollisionLayer(CollisionLayerProjectile)
							// TODO: we should model grenade prop to be something that's kinda 8-gon so that it stops rolling sooner
							projectile.SetCollisionGeometry(unique.Make("Grenade"))
							projectile.SetPhysicsMassOverride(0.1)
							projectile.SetCosmeticOffset(CosmeticOffset{
								Alpha: 2,
								T0:    stx.Now,
								// Ugh. TODO: think how we could make this not as gross.
								Offset: gmath.Matvec3(T_attack.M, gmath.Vec3f32{0.18, 0, -0.2}),
							})
							projectile.SetRenderingGeometry(unique.Make("weapons/grenade_launcher_grenade/geometries/Grenade"))
						})
					}

					stx.Update(weapon, func(stx ScriptContext, weapon Entity) {
						state := weapon.ScriptState().(WeaponGrenadeLauncher)
						defer func() { weapon.SetScriptState(state) }()

						state.CycleEnds = stx.Now.Add(grenadeLauncherStats.CycleDuration)
					})

					for _, prop := range weaponProps {
						stx.Update(prop, func(stx ScriptContext, prop Entity) {
							prop.SetScriptState(Testburger{
								Animation: unique.Make("weapons/grenade_launcher/animations/Fire"),

								PlayTime: stx.Now,

								BaseColor: [4]float32{1, 1, 1, 1},
							})

							prop.SetSoundEffect(SoundEmitter{
								Effect:      unique.Make("weapons/grenade_launcher/sounds/Fire"),
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
