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
	// TODO: make things more interesting
	Chambered bool
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

		Weapon_CreateProp: func(info *UpdateParams, world *World, weapon ecs.ID) Entity2 {
			root := world.CreateEntity(info)
			root.SetScriptState(Testburger{
				BaseColor: [4]float32{1, 1, 1, 1}, // pretend it's a team color
			})
			root.SetSkeleton(unique.Make("weapons/grenade_launcher/skeletons/Armature"))
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
			state := weapon.ScriptState().(WeaponGrenadeLauncher)

			if state.CycleEnds.After(info.Now) {
				return Recoil{}
			}

			// TODO: move this state transition into the normal Think. We'll
			// need to be able to hold onto the player to pull ammo off them.
			if !state.Chambered {
				io.Update(attacker,
					func(info *UpdateParams, mag Entity2, io IO) {
						if mag.Script().Magazine_Pull(info, mag, 0, io) {
							io.Update(weapon,
								func(info *UpdateParams, weapon Entity2, io IO) {
									state := weapon.ScriptState().(WeaponGrenadeLauncher)
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
				io.Create(func(info *UpdateParams, projectile Entity2, io IO) {
					// TODO: don't use prefab here tbh
					// TODO: it would be nice if we could specify this bit without assuming ScriptState type
					projectile.SetScriptState(GrenadeInFlight{
						Attacker:   attacker.ID(),
						LaunchedAt: info.Now,
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
						T0:    info.Now,
						// Ugh. TODO: think how we could make this not as gross.
						Offset: T_attack.M.Mulv(gmath.Vec3f32{0.18, 0, -0.2}),
					})
					projectile.SetRenderingGeometry(unique.Make("weapons/grenade_launcher_grenade/geometries/Grenade"))
				})
			}

			// Apply effects to the props; TODO: let's have scripts on the props
			// instead and let props consult the state.
			for _, id := range weaponProps {
				io.Update(world.GetEntity2(id),
					func(info *UpdateParams, prop Entity2, io IO) {
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
							Effect:      "weapons/grenade_launcher/fire.wav",
							Attenuation: 1,
							PlayTime:    info.Now, // + time.Duration(rng(w.Time, entityID, 0).Int63n(int64(1*time.Millisecond))),
						})
					})
			}

			io.Update(weapon,
				func(info *UpdateParams, weapon Entity2, io IO) {
					state := weapon.ScriptState().(WeaponGrenadeLauncher)
					defer func() { weapon.SetScriptState(state) }()

					state.Chambered = false
					state.CycleEnds = info.Now.Add(grenadeLauncherStats.CycleDuration)
				})

			// TODO: eschew rnd?
			rnd := Rand(info.Now, weapon.ID(), T_attack)

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
