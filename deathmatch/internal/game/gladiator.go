package game

import (
	"math"
	"reflect"
	"slices"
	"time"

	"worldspawn/internal/animgraph"
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
	}

	Vitals struct {
		Health int32

		// TODO: generalize "de/buffs"? We'd have to put them separately from Vitals.

		// How much health we still have to bleed.
		HealthToBleed int32

		// When we lose a health point due to bleed
		NextBleed Time
	}

	Steps float64

	Supported bool

	// FirstPersonCamera is always a descendant of the Character,
	FirstPersonCamera ecs.ID
	// Always a descendant of Character.
	FirstPersonHands ecs.ID

	Weapon ecs.ID

	// The first person prop is always a descendant of FirstPersonHands.
	FirstPersonWeaponProp ecs.ID
	// The third person prop is always a descendant of Character.
	ThirdPersonWeaponProp ecs.ID

	Slots [4]ecs.ID

	// TODO: make Inventory more abstract. What we really wanna do is just try
	// to put items in there and take items in there.
}

func (Gladiator) entity() {}

func init() {
	scripts["gladiator"] = script{
		Type: reflect.TypeFor[Gladiator](),

		Input: func(world *World, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
			gladiator, _ := world.GetEntity[Gladiator](id)
			defer func() { world.Entity.Set(id, gladiator) }()

			var switchToWeapon ecs.ID

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
				if !(0 <= int(cmd) && int(cmd) < len(gladiator.Slots)) {
					break
				}

				switchToWeapon = gladiator.Slots[cmd]

			default:
				// TODO: we should not hit this with nil either
				if cmd != nil {
					panic("unreachable")
				}
			}

			if !world.EntityExists(gladiator.Weapon) && switchToWeapon == 0 {
				for _, slot := range gladiator.Slots {
					if slot != 0 {
						switchToWeapon = slot
						break
					}
				}
			}

			// TODO: rewrite this
			if world.EntityExists(switchToWeapon) && gladiator.Weapon != switchToWeapon {
				// TODO: make weapon switching predicted
				if !info.Speculating {
					gladiator.Weapon = 0

					if world.EntityExists(gladiator.FirstPersonWeaponProp) {
						world.Delete.Set(gladiator.FirstPersonWeaponProp, struct{}{})
					}
					gladiator.ThirdPersonWeaponProp = 0

					if world.EntityExists(gladiator.ThirdPersonWeaponProp) {
						world.Delete.Set(gladiator.ThirdPersonWeaponProp, struct{}{})
					}
					gladiator.FirstPersonWeaponProp = 0

					// Now we can switch the weapons

					if world.EntityExists(switchToWeapon) {
						gladiator.Weapon = switchToWeapon

						// Create the weapon props

						script := world.GetScriptFuncs(switchToWeapon)
						if script.WeaponCreateProp != nil {
							var hint WeaponHint
							if script.WeaponHint != nil {
								hint = script.WeaponHint(world, switchToWeapon)
							}

							gladiator.FirstPersonWeaponProp = script.WeaponCreateProp(world, switchToWeapon, info)
							// TODO: parent it directly to the camera instead.
							world.SetParent(gladiator.FirstPersonWeaponProp, gladiator.FirstPersonHands)
							world.SetTransform(gladiator.FirstPersonWeaponProp, hint.FirstPersonPropTRS)
							world.VisibilityCondition.Set(gladiator.FirstPersonWeaponProp,
								VisibilityCondition{Mask: 0b01, Camera: gladiator.FirstPersonCamera})

							gladiator.ThirdPersonWeaponProp = script.WeaponCreateProp(world, switchToWeapon, info)
							world.SetParent(gladiator.ThirdPersonWeaponProp, id)
							world.ParentBone.Set(gladiator.ThirdPersonWeaponProp, "hand.R")
							world.VisibilityCondition.Set(gladiator.ThirdPersonWeaponProp,
								VisibilityCondition{Mask: 0b10, Camera: gladiator.FirstPersonCamera})
						}
					}
				}
			}

			// TODO: under some conditions we should autoselect a gun for the player

			/*
				if world.EntityExists(gladiator.Weapon) {
					var buttons WeaponButtons
					if gladiator.Input.HeldButtons&uint64(1<<ButtonAttack) != 0 {
						buttons |= WeaponTrigger
					}

					T_attack := world.GetGlobalTransform(id).
						Mul(gmath.TRS3f64{
							T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
							R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
							S: gmath.Mat3x3UOne[float32](),
						}.Compose())

					v_attack, _ := world.Velocity.Get(id)

					// TODO: shooter id we pass should be that of player, actually. Maybe we
					// should have a component to attribute kills and damage to something
					// else.
					stepResult := world.GetScriptFuncs(gladiator.Weapon).WeaponUpdate(world, gladiator.Weapon, []ecs.ID{gladiator.FirstPersonWeaponProp}, id, T_attack, v_attack, buttons, info)
					// TODO: apply some part of the recoil as viewpunch?
					// TODO: make sure we don't overflow LookDir
					gladiator.Input.LookDir[0] += stepResult.Recoil[0]
					gladiator.Input.LookDir[1] += stepResult.Recoil[1]
				}
			*/

			// TODO: factor this out?
			world.SetTransform(gladiator.FirstPersonCamera,
				gmath.TRS3f64{
					T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
					R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
					S: gmath.Mat3x3UOne[float32](),
				})
		},

		Think: func(world *World, entity ecs.ID, info *UpdateParams) {
			// TODO: sort this out pls and make it obey Think rules, i.e. put all mutations into lambdas.

			io := IO{world.Updates, &world.globalUpdates, entity}

			// TODO: this is incredibly gross and ugly, FIXME
			gladiator, _ := world.GetEntity[Gladiator](entity)

			var recoil Recoil
			if world.EntityExists(gladiator.Weapon) {
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

				// TODO: shooter id we pass should be that of player, actually. Maybe we
				// should have a component to attribute kills and damage to something
				// else.
				recoil = world.GetScriptFuncs(gladiator.Weapon).WeaponThink(world, gladiator.Weapon, []ecs.ID{gladiator.FirstPersonWeaponProp}, entity, T_attack, v_attack, buttons, info)
			}

			// TODO: apply some part of the recoil as viewpunch?
			// TODO: make sure we don't overflow LookDir
			gladiator.Input.LookDir[0] += recoil.Recoil[0]
			gladiator.Input.LookDir[1] += recoil.Recoil[1]

			// EXTREMELY YUCKY!!!!!!!!!!!!
			{
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

				world.Pose.Set(entity, pose)
			}

			velocity, _ := world.Velocity.Get(entity)

			io.EnqueueEntityUpdate(gladiator.FirstPersonHands,
				func(world *World, id ecs.ID, updateParams *UpdateParams) {
					world.SetTransform(id,
						gmath.TRS3f64{
							T: gmath.Vec3f64{0, 1, 0}.
								Scale(math.Sin(float64(world.Now.Sub(Time{}))/1e9*6) * 0.03 * min(float64(velocity.Linear.Length()/6), 1)),
							R: gmath.Rot3One(),
							S: gmath.Mat3x3UOne[float32](),
						})
				})

			io.EnqueueEntityUpdate(gladiator.FirstPersonCamera,
				func(world *World, id ecs.ID, updateParams *UpdateParams) {
					world.SetTransform(id,
						gmath.TRS3f64{
							T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
							R: e01.Pow(4 * gladiator.Input.LookDir[0]).Mul(e12.Pow(4 * gladiator.Input.LookDir[1])),
							S: gmath.Mat3x3UOne[float32](),
						})
				})

			trs := world.GetTransform(entity)

			rotation := trs.R.Mul(e01.Pow(4 * gladiator.Input.LookDir[0]))

			move := gladiator.Input.WalkVel
			if lengthSqr := move.Dot(move); lengthSqr > 1 {
				move = move.Scale(1 / float32(math.Sqrt(float64(lengthSqr))))
			}

			localVel := rotation.Inverse().Rotate(velocity.Linear)
			if gladiator.Supported {
				localVel[0] = move[0] * gladiatorStats.WalkSpeed
				localVel[1] = move[1] * gladiatorStats.WalkSpeed
				if gladiator.Input.HeldButtons&(1<<ButtonJump) != 0 {
					localVel[2] = 4
				}
			}
			velocity.Linear = rotation.Rotate(localVel)

			if !gladiator.Supported {
				velocity.Linear = velocity.Linear.Add(world.Globals().Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
			}

			velocity.Linear = gladiator.asdasd(world, entity, velocity.Linear, info.Δt)

			if gladiator.Supported {
				gladiator.Steps += float64(velocity.Linear.Length()) * durationToFloatSeconds(info.Δt)
			}
			if gladiator.Steps > 3 {
				world.SoundEffect.Set(entity, SoundEmitter{
					Effect:      "step.wav",
					Attenuation: 1,
					PlayTime:    world.Now,
				})
				gladiator.Steps = 0
			}

			for gladiator.Vitals.HealthToBleed > 0 && !gladiator.Vitals.NextBleed.After(world.Now) {
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

				world.Delete.Set(entity, struct{}{})
			}

			world.Entity.Set(entity, gladiator)
			world.Velocity.Set(entity, velocity)
		},

		Impact: func(world *World, id ecs.ID, impact Impact, info *UpdateParams) {
			// TODO: be verbose when computing the modifier
			modifier := 1.0
			if id == impact.Attacker {
				modifier /= 2
			}

			// Ceil to avoid rounding to zero
			modifiedDamage := int32(math.Ceil(float64(impact.Damage) * modifier))

			directDamage := int32(math.Ceil(float64(modifiedDamage) * (1 - float64(impactBleedFactor[impact.Type]))))
			bleeding := modifiedDamage - directDamage

			gladiator, _ := world.GetEntity[Gladiator](id)
			gladiator.Vitals.Health -= directDamage
			gladiator.Vitals.HealthToBleed += bleeding
			gladiator.Vitals.NextBleed = world.Now
			world.Entity.Set(id, gladiator)
		},
	}
}

// TODO: pass down the info like team, character, weapons etc. Don't pass Player
// as-is though. Or actually do pass it but keep it optional?
func (world *World) spawnGladiator(T gmath.TRS3f64, info *UpdateParams) ecs.ID {
	gladiator := world.CreateEntity(info)

	camera := world.CreateEntity(info)
	world.SetParent(camera, gladiator)
	// The transform will be set at the next simulation step.

	hands := world.CreateEntity(info)
	world.SetParent(hands, camera)
	world.SetTransform(hands, gmath.TRS3One[float64]())
	world.VisibilityCondition.Set(hands, VisibilityCondition{Mask: 0b01, Camera: camera})
	world.RenderingGeometry.Set(hands, "testcharacter4/geometries/Hands")

	world.SetScript(gladiator, "gladiator")
	s := Gladiator{
		FirstPersonCamera: camera,
		FirstPersonHands:  hands,
	}
	s.Vitals.Health = 100
	world.Entity.Set(gladiator, s)
	world.SetTransform(gladiator, T)
	world.Skeleton.Set(gladiator, "testcharacter4/skeletons/metarig")
	world.CollisionGeometry.Set(gladiator, "Gladiator")
	world.CollisionLayer.Set(gladiator, CollisionLayerMovingKinematic)
	world.PhysicsMassOverride.Set(gladiator, 100)
	world.ShouldSetOffFuseOnImpact.Set(gladiator, struct{}{})
	world.VisibilityCondition.Set(gladiator, VisibilityCondition{Mask: 0b10, Camera: camera})
	world.RenderingGeometry.Set(gladiator, "testcharacter4/geometries/TestCharacter4")

	// Give the gladiator some guns

	{
		weapon := world.CreateEntity(info)
		world.SetScript(weapon, "weapon_grenade_launcher")

		world.GiveWeapon(gladiator, weapon)
	}

	{
		weapon := world.CreateEntity(info)
		world.SetScript(weapon, "weapon_physgun")

		world.GiveWeapon(gladiator, weapon)
	}

	return gladiator
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
	trs := world.GetTransform(id)

	up := gmath.Vec3f32{0, 0, 1}

	var planes []gmath.Vec4f32

	wasSupported := gladiator.Supported

	gladiator.Supported = false

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
				gladiator.Steps = 3
			}
			gladiator.Supported = true
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

// TODO: delete this and handle it entirely in the character's code
func (world *World) GiveWeapon(id ecs.ID, weapon ecs.ID) {
	char := mustOk(world.GetEntity[Gladiator](id))
	freeSlot := slices.Index(char.Slots[:], 0)
	char.Slots[freeSlot] = weapon
	world.Entity.Set(id, char)

	world.SetParent(weapon, id)
}
