package game

import (
	"math"
	"reflect"
	"slices"
	"time"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: turn these into constants and move things into a script
var gladiatorStats = struct {
	StandingHeight     float32
	StandingViewHeight float32 // TODO: I don't really like this existing

	WalkSpeed                float32
	BackwardsWalkSpeedFactor float32
	WalkAcceleration         float32 // TODO: replace with strength
	AirAcceleration          float32
	CosMaxSlopeAngle         float32
	MaxStepHeight            float32
	JumpVelocity             float32
}{
	StandingHeight:     1.9,
	StandingViewHeight: 1.9 - 0.1,

	WalkSpeed:                21.6 / 3.6,
	BackwardsWalkSpeedFactor: 0.8,
	WalkAcceleration:         35,
	JumpVelocity:             4,
}

type Gladiator struct {
	Input struct {
		// Direction we're looking towards. In spherical coordinates following
		// the (e01, e12) convention, in turns.
		//
		// TODO: change this to be fixed point in [0, 1)?
		LookDir [2]float32

		WalkVel gmath.Vec2f32

		HeldButtons uint64

		Slot int8
	}

	Vitals struct {
		Health int32

		// TODO: generalize "de/buffs"? We'd have to put them separately from Vitals.

		// How much health we still have to bleed.
		HealthToBleed int32

		// When we lose a health point due to bleed
		NextBleed Time
	}

	// TODO: factor out movement into its own struct? Yeah
	Motion struct {
		Steps float64

		Supported bool
	}

	// Always a descendant of the Character,
	FirstPersonCamera ecs.ID

	// Always a descendant of Character.
	FirstPersonHands ecs.ID

	HeldWeapon struct {
		Entity ecs.ID

		// TODO: should we have a prop per weapon all the time? Or create them
		// when we switch to the weapon. Having props all the time would be
		// kinda bad.

		// The first person prop (0) is always a descendant of FirstPersonHands.
		// The third person prop (1) is always a descendant of Gladiator.
		//
		// TODO: define a enum instead of writing indices out manually
		// TODO: it should be parented to camera instead of hands. We'll use IK
		// to position hands where we need things to be.
		Props [2]ecs.ID
	}

	// TODO: put it into a proper struct
	Inventory struct {
		Slots [4]ecs.ID

		Ammo [10]int8
	}
}

func init() {
	Scripts[reflect.TypeFor[Gladiator]()] = script{
		Input: func(info *UpdateParams, world *World, id ecs.ID, cmd TimestampedInputCmd) {
			gladiator, _ := world.GetEntity[Gladiator](id)
			defer func() { world.Entity.Set(id, gladiator) }()

			switch cmd := cmd.Cmd.(type) {
			case InputCmdDLookXY:
				gladiator.Input.LookDir[0] = float32(math.Mod(float64(gladiator.Input.LookDir[0]-float32(cmd)), 1))
			case InputCmdDLookYZ:
				gladiator.Input.LookDir[1] = min(max(gladiator.Input.LookDir[1]-float32(cmd), -0.25), 0.25)
			case InputCmdMoveX:
				gladiator.Input.WalkVel[0] = float32(cmd)
			case InputCmdMoveY:
				gladiator.Input.WalkVel[1] = float32(cmd)
			case InputCmdPressButton:
				gladiator.Input.HeldButtons |= uint64(1) << cmd
			case InputCmdReleaseButton:
				gladiator.Input.HeldButtons &^= uint64(1) << cmd
			case Slot:
				gladiator.Input.Slot = int8(cmd)

			default:
				// TODO: we should not hit this with nil either
				if cmd != nil {
					panic("unreachable")
				}
			}

			// TODO: autoselect the gun here

			switchToWeaponID := gladiator.Inventory.Slots[gladiator.Input.Slot] // TODO: be safe with the values Slot might have

			// TODO: make weapon switching predicted
			if !info.Speculating && gladiator.HeldWeapon.Entity != switchToWeaponID {
				for _, propID := range gladiator.HeldWeapon.Props {
					if prop := world.GetEntity2(propID); prop.Valid() {
						prop.MarkForDeletion()
					}
				}
				clear(gladiator.HeldWeapon.Props[:])

				// Now we can switch the weapons

				gladiator.HeldWeapon.Entity = switchToWeaponID

				if newWeapon := world.GetEntity2(gladiator.HeldWeapon.Entity); newWeapon.Valid() {
					// We've switched to the new weapon, create the new weapon props

					script := newWeapon.Script()
					if script.Weapon_CreateProp != nil {
						hint := script.Weapon_Hint(info, world, newWeapon.ID())

						for i := range 2 {
							prop := script.Weapon_CreateProp(info, world, newWeapon.ID())

							switch i {
							case 0:
								// TODO: parent it directly to the camera instead.
								prop.SetParent(gladiator.FirstPersonHands)
								prop.SetTransform(hint.FirstPersonPropTransform)
								prop.SetVisibilityCondition(VisibilityCondition{Mask: 0b01, Camera: gladiator.FirstPersonCamera})

							case 1:
								prop.SetParent(id)
								prop.SetParentBone(unique.Make("hand.R"))
								prop.SetTransform(gmath.TRS3One[float64]())
								prop.SetVisibilityCondition(VisibilityCondition{Mask: 0b10, Camera: gladiator.FirstPersonCamera})
							}

							gladiator.HeldWeapon.Props[i] = prop.ID()
						}
					}
				}
			}

			// TODO: factor this out?
			world.GetEntity2(gladiator.FirstPersonCamera).
				SetTransform(gmath.TRS3f64{
					T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
					R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
					S: gmath.Mat3x3UOne[float32](),
				})
		},

		Think: func(info *UpdateParams, world *World, entity ecs.ID, io IO) {
			// TODO: sort this out pls and make it obey Think rules, i.e. put all mutations into lambdas.

			gladiator, _ := world.GetEntity[Gladiator](entity)

			var recoil Recoil
			if weapon := world.GetEntity2(gladiator.HeldWeapon.Entity); weapon.Valid() {
				var buttons WeaponButtons
				if gladiator.Input.HeldButtons&uint64(1<<ButtonAttack) != 0 {
					buttons |= WeaponTrigger
				}

				T_attack := world.GetGlobalTransform(entity).
					Mul(gmath.TRS3f64{
						T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
						R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
						S: gmath.Mat3x3UOne[float32](),
					}.Compose())

				v_attack, _ := world.Velocity.Get(entity)

				recoil = weapon.Script().Weapon_Think(info, world, weapon.id, gladiator.HeldWeapon.Props[:], entity, T_attack, v_attack, buttons, io)
			}

			// EXTREMELY YUCKY!!!!!!!!!!!!
			/*
				{
					// TODO: interpolate between two poses instead

					skelly := world.GetSkeleton(entity)

					localTransforms := map[int]gmath.Affine3f32{}
					localTransforms[skelly.JointByName("spine")] =
						skelly.BindPoseInverse[skelly.JointByName("spine")].
							Mul(gmath.TRS3f32{
								R: gmath.Rot3AToB(gmath.Vec3f32{1, 0, 0}, gmath.Vec3f32{0, 1, 0}).
									Pow(4 * gladiator.Input.LookDir[0]),
								S: gmath.Mat3x3UOne[float32](),
							}.Compose()).
							Mul(skelly.BindPose[skelly.JointByName("spine")])
					localTransforms[skelly.JointByName("spine.001")] =
						gmath.TRS3f32{
							R: gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{0, 1, 0}).
								Pow(4 * gladiator.Input.LookDir[1]),
							S: gmath.Mat3x3UOne[float32](),
						}.Compose()

					pose := animgraph.Pose{
						Bones: map[int]gmath.Affine3f32{},
					}

					// TODO: flooding would be more efficient
					for bone := range skelly.JointNames {
						A := gmath.Affine3One[float32]()

						tmp := bone
						for {
							B, ok := localTransforms[tmp]
							if !ok {
								B = gmath.Affine3One[float32]()
							}

							A = skelly.ParentRelative[tmp].Mul(B).Mul(A)

							parent := skelly.Parent[tmp]
							if parent == -1 {
								break
							}
							tmp = parent
						}

						pose.Bones[bone] = A.Mul(skelly.BindPoseInverse[bone])
					}

					io.EnqueueEntityUpdate(entity, func(world *World, entity ecs.ID, info *UpdateParams) {
						world.Pose.Set(entity, pose)
					})
				}
			*/

			// TODO: redo sway animation
			{
				velocity, _ := world.Velocity.Get(entity)

				io.EnqueueEntityUpdate(gladiator.FirstPersonHands,
					func(_ *UpdateParams, hands Entity2, io IO) {
						hands.SetTransform(gmath.TRS3f64{
							T: gmath.Vec3f64{0, 1, 0}.
								Scale(math.Sin(float64(io.world.Now.Sub(Time{}))/1e9*6) * 0.03 * min(float64(velocity.Linear.Length()/6), 1)),
							R: gmath.Rot3One(),
							S: gmath.Mat3x3UOne[float32](),
						})
					})
			}

			io.EnqueueEntityUpdate(gladiator.FirstPersonCamera,
				func(_ *UpdateParams, camera Entity2, io IO) {
					camera.SetTransform(gmath.TRS3f64{
						T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
						R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
						S: gmath.Mat3x3UOne[float32](),
					})
				})

			io.EnqueueEntityUpdate(entity,
				func(_ *UpdateParams, entity Entity2, io IO) {
					gladiator, _ := entity.ScriptState().(Gladiator)
					defer func() { entity.SetScriptState(gladiator) }()

					// TODO: apply some part of the recoil as viewpunch?
					// TODO: make sure we don't overflow LookDir
					gladiator.Input.LookDir[0] += recoil.Recoil[0]
					gladiator.Input.LookDir[1] += recoil.Recoil[1]

					velocity := entity.Velocity()

					trs := entity.Transform()

					rotation := trs.R.Mul(e01.Pow(4 * gladiator.Input.LookDir[0]))

					move := gladiator.Input.WalkVel
					if lengthSqr := move.Dot(move); lengthSqr > 1 {
						move = move.Scale(1 / float32(math.Sqrt(float64(lengthSqr))))
					}

					v_local := rotation.Inverse().Rotate(velocity.Linear)
					if gladiator.Motion.Supported {
						v_local[0] = move[0] * gladiatorStats.WalkSpeed
						v_local[1] = move[1] * gladiatorStats.WalkSpeed
						if gladiator.Input.HeldButtons&(1<<ButtonJump) != 0 {
							v_local[2] = 4
						}
					}
					velocity.Linear = rotation.Rotate(v_local)
					if !gladiator.Motion.Supported {
						velocity.Linear = velocity.Linear.Add(io.world.Globals().Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
					}
					velocity.Linear = gladiator.asdasd(io.world, entity.id, velocity.Linear, info.Δt)

					if gladiator.Motion.Supported {
						gladiator.Motion.Steps += float64(velocity.Linear.Length()) * durationToFloatSeconds(info.Δt)
					}
					if gladiator.Motion.Steps > 3 {
						entity.SetSoundEffect(SoundEmitter{
							Effect:      "step.wav",
							Attenuation: 1,
							PlayTime:    io.world.Now,
						})
						gladiator.Motion.Steps = 0
					}

					for gladiator.Vitals.HealthToBleed > 0 && !gladiator.Vitals.NextBleed.After(io.world.Now) {
						const bleedSpeedFactor = 0.1
						const minBleedSpeed = 1.0

						gladiator.Vitals.Health--
						gladiator.Vitals.HealthToBleed--
						gladiator.Vitals.NextBleed = gladiator.Vitals.NextBleed.
							Add(time.Duration(1e9 / max(float64(gladiator.Vitals.HealthToBleed)*bleedSpeedFactor, minBleedSpeed)))
					}

					// TODO: we should probably enqueue the death so that it happens
					// after all impacts. That way we can see how far we are below 0
					// and therefore decide whether we want to spawn gibs or just
					// drop a ragdoll.
					if gladiator.Vitals.Health <= 0 {
						info.Logger.Info("killing myself!!!", "id", entity)

						// TODO: spawn ragdoll or gibs

						// TODO: bump the death counter and also attribute kill and/or assist to
						// other players. We'll need to keep a damage log for that (even if
						// limited)

						entity.MarkForDeletion()
					}

					entity.SetVelocity(velocity)
				})
		},

		Impact: func(info *UpdateParams, gladiator Entity2, impact Impact, io IO) {
			// TODO: be verbose when computing the modifier
			modifier := 1.0
			if gladiator.id == impact.Attacker {
				modifier /= 2
			}

			// Ceil to avoid rounding to zero
			modifiedDamage := int32(math.Ceil(float64(impact.Damage) * modifier))

			directDamage := int32(math.Ceil(float64(modifiedDamage) * (1 - float64(impactBleedFactor[impact.Type]))))
			bleeding := modifiedDamage - directDamage

			gladiator.UpdateScriptState(func(state *Gladiator) {
				state.Vitals.Health -= directDamage
				state.Vitals.HealthToBleed += bleeding
				state.Vitals.NextBleed = io.world.Now
			})
		},

		Magazine_Pull: func(info *UpdateParams, entity Entity2, ammoType AmmoType, io IO) bool {
			state, _ := entity.ScriptState().(Gladiator)
			if state.Inventory.Ammo[ammoType] > 0 {
				state.Inventory.Ammo[ammoType]--
				entity.world.Entity.Set(entity.id, state)
				return true
			}
			return false
		},
	}
}

// TODO: pass down the info like team, character, weapons etc. Don't pass Player
// as-is though. Or actually do pass it but keep it optional?
func (world *World) spawnGladiator(T gmath.TRS3f64, info *UpdateParams) ecs.ID {
	gladiator := world.CreateEntity(info)
	gladiator.SetTransform(T)
	gladiator.SetSkeleton(unique.Make("testcharacter4/skeletons/metarig"))
	gladiator.SetCollisionLayer(CollisionLayerMovingKinematic)
	gladiator.SetCollisionGeometry(unique.Make("Gladiator"))
	gladiator.SetPhysicsMassOverride(100)
	gladiator.SetShouldSetOffFuseOnImpact(true)
	gladiator.SetRenderingGeometry(unique.Make("testcharacter4/geometries/TestCharacter4"))

	camera := world.CreateEntity(info)
	camera.SetParent(gladiator.id)
	gladiator.SetVisibilityCondition(VisibilityCondition{Mask: 0b10, Camera: camera.id})
	// The transform will be set at the next simulation step.

	hands := world.CreateEntity(info)
	hands.SetParent(camera.id)
	hands.SetTransform(gmath.TRS3One[float64]())
	hands.SetVisibilityCondition(VisibilityCondition{Mask: 0b01, Camera: camera.id})
	hands.SetRenderingGeometry(unique.Make("testcharacter4/geometries/Hands"))

	s := Gladiator{
		FirstPersonCamera: camera.id,
		FirstPersonHands:  hands.id,
	}
	s.Vitals.Health = 100
	// TODO: define loadout somehow better so that ammo pickup knows what to do
	s.Inventory.Ammo[0] = 10
	world.Entity.Set(gladiator.id, s)

	// Give the gladiator some guns

	{
		weapon := world.CreateEntity(info)
		world.Entity.Set(weapon.id, WeaponGrenadeLauncher{})

		world.GiveWeapon(gladiator.id, weapon.id)
	}

	{
		weapon := world.CreateEntity(info)
		world.Entity.Set(weapon.id, WeaponPhysgun{})

		world.GiveWeapon(gladiator.id, weapon.id)
	}

	return gladiator.id
}

func planeNormal(plane gmath.Vec4f32) gmath.Vec3f32 {
	return gmath.Vec3f32{plane[0], plane[1], plane[2]}
}

func planeSignedDistance(plane gmath.Vec4f32, point gmath.Vec3f32) float32 {
	return point.Dot(planeNormal(plane)) + plane[3]
}

// TODO: introduce typedefs to physics for query types
// TODO: replace query pipelines with a pile of funcs? Maybe we could even do
// func vararg options. Ugh...

type gladiatorMovementQueryPipeline struct {
	player physics.BodyID
	hits   []physics.SceneIntersection[physics.OverlapHit]
}

func (*gladiatorMovementQueryPipeline) FilterLayer(layer int) bool {
	// TOOD: we should just poke the rule table
	return layer == int(CollisionLayerNonMoving) || layer == int(CollisionLayerMoving)
}

func (a *gladiatorMovementQueryPipeline) FilterBody(body physics.BodyID) bool {
	return body != a.player
}

func (a *gladiatorMovementQueryPipeline) Hit(x physics.SceneIntersection[physics.OverlapHit]) physics.QueryPipelineControl {
	a.hits = append(a.hits, x)
	return physics.IgnoreHit
}

func (gladiator *Gladiator) asdasd(world *World, id ecs.ID, velocity gmath.Vec3f32, Δt time.Duration) gmath.Vec3f32 {
	trs := world.GetEntity2(id).Transform()

	up := gmath.Vec3f32{0, 0, 1}

	var planes []gmath.Vec4f32

	wasSupported := gladiator.Motion.Supported

	gladiator.Motion.Supported = false

	contactBuffer := gladiatorMovementQueryPipeline{
		player: physics.BodyID(id.Index()),
	}
	world.physics.Overlap(
		physics.Overlap{
			Pos:                   trs.T,
			Rot:                   trs.R,
			Scale:                 gmath.Vec3Ones[float32](),
			Shape:                 getShape(world, id),
			MovementDirection:     velocity.Normalize(),
			MaxSeparationDistance: 0.1,
		},
		&contactBuffer)

	// TODO: we could move this into the pipeline thingy
	for _, contact := range contactBuffer.hits {
		normal := contact.Geometry.PenetrationAxis.Normalize().Scale(-1)
		if normal.Dot(up) < 0.7 {
			if false {
				// This prevents us from walking up steep ramps, but has an issue in
				// that sometimes we get a ghost steep ramp
				normal2 := normal.Cross(up).Normalize().Cross(up).Scale(-1)
				planes = append(planes, gmath.Vec4f32{
					normal2[0],
					normal2[1],
					normal2[2],
					-contact.Geometry.PenetrationDepth + 0.1,
				})
			}
		} else {
			if !wasSupported {
				gladiator.Motion.Steps = 3
			}
			gladiator.Motion.Supported = true
		}
		planes = append(planes, gmath.Vec4f32{
			normal[0],
			normal[1],
			normal[2],
			-contact.Geometry.PenetrationDepth + 0.1,
		})
	}

	// TODO: on ground detection

	for i, plane := range planes {
		_ = i

		projectedVelocity := -planeNormal(plane).Dot(velocity)

		if projectedVelocity > 0 {
			toi := -planeSignedDistance(plane, velocity.Scale(float32(durationToFloatSeconds(Δt)))) / projectedVelocity
			if toi <= float32(durationToFloatSeconds(Δt)) {
				velocity = velocity.Sub(planeNormal(plane).Scale(-projectedVelocity))
			}
		}
	}

	const minSpeed float32 = 1e-6

	if velocity.Dot(velocity) < minSpeed*minSpeed {
		velocity = gmath.Vec3f32{}
	}

	return velocity
}

// TODO: delete this
func (world *World) GiveWeapon(id ecs.ID, weapon ecs.ID) {
	char := mustOk(world.GetEntity[Gladiator](id))
	freeSlot := slices.Index(char.Inventory.Slots[:], 0)
	char.Inventory.Slots[freeSlot] = weapon
	world.Entity.Set(id, char)

	world.SetParent(weapon, id)
}
