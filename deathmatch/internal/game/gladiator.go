package game

import (
	"math"
	"reflect"
	"slices"
	"time"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
	"worldspawn/internal/physics"
)

// TODO: turn these into constants
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

	WalkSpeed:                10,
	BackwardsWalkSpeedFactor: 0.8,
	WalkAcceleration:         35,
	JumpVelocity:             8,
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
		Health float32
	}

	// TODO: factor out movement into its own struct? Yeah
	Motion struct {
		Steps float64

		Supported bool
	}

	// Always a descendant of the Character,
	FirstPersonCamera ecs.ID

	// Always a descendant of Character.
	// TODO: rename to HandsFirstPersonProp
	FirstPersonHands ecs.ID

	HeldWeapon struct {
		State           int8 // 0=idle, 1=drawing, 2=hiding
		StateTransition Time

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
			entity := world.Entity(id)

			gladiator := entity.ScriptState().(Gladiator)
			defer func() { entity.SetScriptState(gladiator) }()

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
			case InputCmdSwitchWeapon:
				gladiator.Input.Slot = int8(cmd)

			default:
				// TODO: we should not hit this with nil either
				if cmd != nil {
					panic("unreachable")
				}
			}

			// TODO: factor this out?
			world.Entity(gladiator.FirstPersonCamera).
				SetTransform(gmath.TRS3f64{
					T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
					R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
					S: gmath.Mat3x3UOne[float32](),
				})
		},

		Think: func(stx ScriptContext, world *World, gladiator Entity) {
			state := gladiator.ScriptState().(Gladiator)

			if weapon := world.Entity(state.HeldWeapon.Entity); weapon.IsValid() {
				var props [len(state.HeldWeapon.Props)]Entity
				for i, propID := range state.HeldWeapon.Props {
					props[i] = world.Entity(propID)
				}

				T_attack := world.GetGlobalTransform2(gladiator).
					Mul(gmath.TRS3f64{
						T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
						R: e01.Pow(4 * state.Input.LookDir[0]).Mul(e12.Pow(4 * state.Input.LookDir[1])),
						S: gmath.Mat3x3UOne[float32](),
					}.Affine())
				v_attack := gladiator.Velocity()

				var buttons WeaponButtons
				if state.HeldWeapon.State == 0 {
					if state.Input.HeldButtons&uint64(1<<ButtonAttack) != 0 {
						buttons |= WeaponTrigger
					}
				}

				recoil := func(stx ScriptContext, recoil [2]float32) {
					// TODO: implement
				}

				weapon.Script().Weapon_Think(stx, world, weapon, props[:], gladiator, T_attack, v_attack, buttons, recoil)
			}

			stx.Update(gladiator,
				func(stx ScriptContext, gladiator Entity) {
					state := gladiator.ScriptState().(Gladiator)
					defer func() { gladiator.SetScriptState(state) }()

					// TODO: be safe with the values Slot might have
					switchToWeapon := world.Entity(state.Inventory.Slots[state.Input.Slot])

					if state.HeldWeapon.StateTransition.Compare(stx.Now) <= 0 {
						switch state.HeldWeapon.State {
						case 2:
							for _, propID := range state.HeldWeapon.Props {
								if prop := world.Entity(propID); prop.IsValid() {
									stx.Update(prop, func(stx ScriptContext, prop Entity) { prop.MarkForDeletion() })
								}
							}
							clear(state.HeldWeapon.Props[:])

							// BUG: we can only really query Weapon_Hint inside a
							// Think. I guess we could make Weapon_Hint be a
							// non-thinker, that would be pretty nice I think.
							weaponScript := switchToWeapon.Script()
							hint := weaponScript.Weapon_Hint(stx.UpdateParams, world, switchToWeapon.ID())
							state.HeldWeapon.Entity = switchToWeapon.ID()
							state.HeldWeapon.State = 1
							drawDuration := time.Duration(float64(weaponBaseDrawDuration) * float64(hint.DrawDurationMultiplier))
							state.HeldWeapon.StateTransition = stx.Now.Add(drawDuration)

							if !stx.Speculating && weaponScript.Weapon_CreateProp != nil {
								for i := range 2 {
									weaponScript.Weapon_CreateProp(stx, switchToWeapon, func(stx ScriptContext, prop Entity) {
										switch i {
										case 0:
											// TODO: parent it directly to the camera instead.
											prop.SetParent(state.FirstPersonHands)
											prop.SetTransform(hint.FirstPersonPropTransform)
											prop.SetVisibilityCondition(VisibilityCondition{Mask: 0b01, Camera: state.FirstPersonCamera})

										case 1:
											prop.SetParent(gladiator.ID())
											prop.SetParentBone(unique.Make("hand.R"))
											prop.SetTransform(gmath.TRS3One[float64]())
											prop.SetVisibilityCondition(VisibilityCondition{Mask: 0b10, Camera: state.FirstPersonCamera})
										}

										stx.Update(gladiator, func(stx ScriptContext, entity Entity) {
											state := entity.ScriptState().(Gladiator)
											defer func() { entity.SetScriptState(state) }()

											state.HeldWeapon.Props[i] = prop.ID()
										})
									})
								}
							}

						case 1:
							state.HeldWeapon.State = 0

						case 0:
							if state.HeldWeapon.Entity != switchToWeapon.ID() {
								state.HeldWeapon.State = 2
								if heldWeapon := world.Entity(state.HeldWeapon.Entity); heldWeapon.IsValid() {
									hint := heldWeapon.Script().Weapon_Hint(stx.UpdateParams, world, switchToWeapon.ID())
									hideDuration := time.Duration(float64(weaponBaseHideDuration) * float64(hint.HideDurationMultiplier))
									state.HeldWeapon.StateTransition = stx.Now.Add(hideDuration)
								}
							}
						}
					}

					T := gladiator.Transform()

					v := gladiator.Velocity()
					defer func() { gladiator.SetVelocity(v) }()

					rotation := T.R.Mul(e01.Pow(4 * state.Input.LookDir[0]))

					move := state.Input.WalkVel
					if lengthSqr := move.Dot(move); lengthSqr > 1 {
						move = move.Scale(1 / float32(math.Sqrt(float64(lengthSqr))))
					}

					v_local := rotation.Inv().Rotate(v.Linear)
					if state.Motion.Supported {
						v_local[0] = move[0] * gladiatorStats.WalkSpeed
						v_local[1] = move[1] * gladiatorStats.WalkSpeed
						if state.Input.HeldButtons&(1<<ButtonJump) != 0 {
							v_local[2] = 4
						}
					}
					v.Linear = rotation.Rotate(v_local)
					if !state.Motion.Supported {
						v.Linear = v.Linear.Add(stx.Gravity.Scale(float32(durationToFloatSeconds(stx.Δt))))
					}
					v.Linear = state.asdasd(stx.world, gladiator.ID(), v.Linear, stx.Δt)

					if state.Motion.Supported {
						state.Motion.Steps += float64(v.Linear.Length()) * durationToFloatSeconds(stx.Δt)
					}
					if state.Motion.Steps > 3 {
						gladiator.SetSoundEffect(SoundEmitter{
							Effect:      unique.Make("Step"),
							Attenuation: 1,
							PlayTime:    stx.Now,
						})
						state.Motion.Steps = 0
					}

					// TODO: we should probably enqueue the death so that it happens
					// after all impacts. That way we can see how far we are below 1
					// and therefore decide whether we want to spawn gibs or just
					// drop a ragdoll.
					if state.Vitals.Health <= 0 {
						world.logger.Info("killing myself!!!", "id", gladiator)

						// TODO: spawn ragdoll or gibs

						// TODO: bump the death counter and also attribute kill and/or assist to
						// other players. We'll need to keep a damage log for that (even if
						// limited)

						// TODO: don't delete ourselves actually. Let's have
						// functioning revival instead. Deleting ourselves also
						// interferes with damage attribution.
						gladiator.MarkForDeletion()
					}

					stx.Update(world.Entity(state.FirstPersonCamera), func(stx ScriptContext, camera Entity) {
						camera.SetTransform(gmath.TRS3f64{
							T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
							R: e01.Pow(4 * state.Input.LookDir[0]).Mul(e12.Pow(4 * state.Input.LookDir[1])),
							S: gmath.Mat3x3UOne[float32](),
						})
					})

					{

						stx.Update(world.Entity(state.FirstPersonHands),
							func(stx ScriptContext, hands Entity) {
								hands.SetTransform(gmath.TRS3f64{
									T: gmath.Vec3f64{0, 1, 0}.
										Scale(math.Sin(float64(stx.Now.Sub(Time{}))/1e9*6) * 0.03 * min(float64(v.Linear.Length()/6), 1)),
									R: gmath.Rot3One(),
									S: gmath.Mat3x3UOne[float32](),
								})
							})
					}

					{
						anim := animationCache.Get(unique.Make("testcharacter4/animations/look"))

						// TODO: introduce affine1 helper so that we can do remaps from one range to another

						v := make([]float32, len(anim.Channels()))
						animation.SampleNormalized(anim, 0.5+2*state.Input.LookDir[1], v)

						sk := skeletonCache.Get(gladiator.Skeleton())

						localTransforms := make([]gmath.Affine3f32, sk.NumJoints())
						for i := range localTransforms {
							localTransforms[i] = gmath.Affine3One[float32]()
						}

						poseMapperCache.Get(poseMapperKey{anim, sk})(v, localTransforms)

						b_spine := sk.JointByName(unique.Make("spine"))

						localTransforms[b_spine] =
							sk.BindPoseInv[b_spine].
								Mul(gmath.TRS3f32{
									R: gmath.Rot3AToB(gmath.Vec3f32{1, 0, 0}, gmath.Vec3f32{0, 1, 0}).
										Pow(4 * state.Input.LookDir[0]),
									S: gmath.Mat3x3UOne[float32](),
								}.Affine()).
								Mul(sk.BindPose[b_spine]).
								Mul(localTransforms[b_spine])

						pose := make(skeleton.Pose, sk.NumJoints())
						sk.ForwardKinematics(localTransforms, pose)

						gladiator.SetPose(pose)
					}
				})
		},

		Impact: func(stx ScriptContext, gladiator Entity, impact Impact) {
			// TODO: be verbose when computing the modifier
			modifier := float32(1.0)
			if gladiator == impact.Attacker {
				modifier /= 2
			}

			impact.Damage *= modifier

			state := gladiator.ScriptState().(Gladiator)
			defer func() { gladiator.SetScriptState(state) }()

			state.Vitals.Health -= impact.Damage
		},

		Magazine_Pull: func(stx ScriptContext, entity Entity, ammoType AmmoType, minAmount, maxAmount int) int {
			state := entity.ScriptState().(Gladiator)
			if state.Inventory.Ammo[ammoType] <= int8(minAmount) {
				return 0
			}
			defer func() { entity.SetScriptState(state) }()

			amount := int8(min(int(state.Inventory.Ammo[ammoType]), maxAmount))
			state.Inventory.Ammo[ammoType] -= amount
			return int(amount)
		},
	}
}

// TODO: rewrite this so that it uses IO.Create
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
	camera.SetParent(gladiator.ID())
	gladiator.SetVisibilityCondition(VisibilityCondition{Mask: 0b10, Camera: camera.ID()})
	// The transform will be set at the next tick.

	hands := world.CreateEntity(info)
	hands.SetParent(camera.ID())
	hands.SetTransform(gmath.TRS3One[float64]())
	hands.SetVisibilityCondition(VisibilityCondition{Mask: 0b01, Camera: camera.ID()})
	hands.SetRenderingGeometry(unique.Make("testcharacter4/geometries/Hands"))

	s := Gladiator{
		FirstPersonCamera: camera.ID(),
		FirstPersonHands:  hands.ID(),
	}
	s.Vitals.Health = 100
	// TODO: define loadout somehow better so that ammo pickup knows what to do
	s.Inventory.Ammo[0] = 10
	s.Inventory.Ammo[1] = 100
	gladiator.SetScriptState(s)

	// Give the gladiator some guns
	//
	// TODO: this should depend on

	{
		weapon := world.CreateEntity(info)
		weapon.SetScriptState(WeaponGrenadeLauncher{})

		world.GiveWeapon(gladiator, weapon)
	}

	{
		weapon := world.CreateEntity(info)
		weapon.SetScriptState(WeaponAssaultRifle{})

		world.GiveWeapon(gladiator, weapon)
	}

	{
		weapon := world.CreateEntity(info)
		weapon.SetScriptState(WeaponPhysgun{})

		world.GiveWeapon(gladiator, weapon)
	}

	return gladiator.ID()
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
	return layer == int(CollisionLayerBackground) || layer == int(CollisionLayerProp)
}

func (a *gladiatorMovementQueryPipeline) FilterBody(body physics.BodyID) bool {
	return body != a.player
}

func (a *gladiatorMovementQueryPipeline) Hit(x physics.SceneIntersection[physics.OverlapHit]) physics.QueryPipelineControl {
	a.hits = append(a.hits, x)
	return physics.IgnoreHit
}

func (gladiator *Gladiator) asdasd(world *World, id ecs.ID, velocity gmath.Vec3f32, Δt time.Duration) gmath.Vec3f32 {
	trs := world.Entity(id).Transform()

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
			Shape:                 getShape(Entity{&world.Columns, id}),
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
func (world *World) GiveWeapon(entity Entity, weapon Entity) {
	char := entity.ScriptState().(Gladiator)
	freeSlot := slices.Index(char.Inventory.Slots[:], 0)
	char.Inventory.Slots[freeSlot] = weapon.ID()
	entity.SetScriptState(char)

	weapon.SetParent(entity.ID())
}
