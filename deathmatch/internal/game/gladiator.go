package game

import (
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: the users of this should be factored out into a function

var planeXY = gmath.Plane3OnVectors(gmath.Vec3f32{1, 0, 0}, gmath.Vec3f32{0, 1, 0})
var planeYZ = gmath.Plane3OnVectors(gmath.Vec3f32{0, 1, 0}, gmath.Vec3f32{0, 0, 1})

var gladiatorStats = struct {
	StandingHeight     float32
	StandingViewHeight float32 // TODO: I don't really like this existing

	WalkVelocity                float32
	BackwardsWalkVelocityFactor float32
	WalkAcceleration            float32
	AirAcceleration             float32
	CosMaxSlopeAngle            float32
	MaxStepHeight               float32
	JumpVelocity                float32
}{
	StandingHeight:     1.9,
	StandingViewHeight: 1.9 - 0.1,

	WalkVelocity:                21.6 / 3.6,
	BackwardsWalkVelocityFactor: 0.8,
	WalkAcceleration:            35,
	JumpVelocity:                4,
}

type Gladiator struct {
	Health int32

	LookDir     [2]float32 // spherical unit vec3; TODO: swap coordinates so that polar angle is [0] and flip its sign
	MoveVec     gmath.Vec2f32
	HeldButtons uint64

	// TODO: viewshake and viewpunch

	Steps float64

	Supported bool

	// FirstPersonCamera is always a descendant of the Character,
	FirstPersonCamera ecs.ID
	// Always a descendant of Character.
	FirstPersonHands ecs.ID

	ActiveWeapon ecs.ID // TODO: rename to HeldWeapon?
	// The first person prop is always a descendant of FirstPersonHands.
	ActiveWeaponFirstPersonProp ecs.ID
	// The third person prop is always a descendant of Character.
	ActiveWeaponThirdPersonProp ecs.ID

	Slots [4]ecs.ID
}

var _ PlayableCharacter = Gladiator{}

func (Gladiator) entity() {}

// TODO: pass down the info like team, character, weapons etc. Don't pass Player
// as-is though.
func createGladiator(scene *Scene, info *UpdateParams) ecs.ID {
	// TODO: probably move this out of here? What if we want to spawn the
	// gladiator elsewhere (e.g. next to teammates.) The whole logic should
	// probably be delegated elsewhere.
	var playerSpawns []ecs.ID
	for id, entity := range ecs.All(&scene.Entity) {
		if _, ok := entity.(PlayerSpawn); ok {
			playerSpawns = append(playerSpawns, id)
		}
	}
	// TODO: we need a special rand utility so that random sequences are
	// reproducible
	T := scene.GetGlobalTransform(playerSpawns[rand.IntN(len(playerSpawns))])

	gladiator := scene.CreateEntity(info)

	camera := scene.CreateEntity(info)
	scene.SetParent(camera, gladiator)
	// The transform will be set at the next simulation step.

	hands := scene.CreateEntity(info)
	scene.SetParent(hands, camera)
	scene.SetTransform(hands, gmath.TRS3One[float64]())
	scene.VisibilityMask.Set(gladiator, VisibilityMask{Mask: 0b01, Camera: camera})
	scene.RenderingGeometry.Set(hands, "testcharacter4/geometries/Hands")

	scene.SetTransform(gladiator, T.TRS())
	scene.Skeleton.Set(gladiator, "testcharacter4/skeletons/metarig")
	scene.CollisionGeometry.Set(gladiator, "FPSCharacter")
	scene.CollisionLayer.Set(gladiator, CollisionLayerMovingKinematic)
	scene.PhysicsMassOverride.Set(gladiator, 100)
	scene.VisibilityMask.Set(gladiator, VisibilityMask{Mask: 0b10, Camera: camera})
	scene.RenderingGeometry.Set(gladiator, "testcharacter4/geometries/TestCharacter4")
	scene.Entity.Set(gladiator, Gladiator{
		FirstPersonCamera: camera,
		FirstPersonHands:  hands,
	})

	// Give the gladiator some guns

	{
		weapon := scene.CreateEntity(info)
		scene.Entity.Set(weapon, WeaponGrenadeLauncher{})

		scene.GiveWeapon(gladiator, weapon)
	}

	{
		weapon := scene.CreateEntity(info)
		scene.Entity.Set(weapon, WeaponSniperRifle{})

		scene.GiveWeapon(gladiator, weapon)
	}

	return gladiator
}

func (gladiator Gladiator) CharacterSubstep(w *Scene, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	defer func() { w.Entity.Set(id, gladiator) }()

	var switchToWeapon ecs.ID

	switch cmd := cmd.Cmd.(type) {
	case InputCmdDLookX:
		gladiator.LookDir[0] = float32(math.Mod(float64(gladiator.LookDir[0]-float32(cmd)), 1))
	case InputCmdDLookY:
		gladiator.LookDir[1] = min(max(gladiator.LookDir[1]-float32(cmd), -0.25), 0.25)
	case InputCmdMoveX:
		gladiator.MoveVec[0] = float32(cmd)
	case InputCmdMoveY:
		gladiator.MoveVec[1] = float32(cmd)
	case InputCmdPressButton:
		gladiator.HeldButtons |= uint64(1) << cmd
	case InputCmdReleaseButton:
		gladiator.HeldButtons &^= uint64(1) << cmd
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

	if !w.EntityExists(gladiator.ActiveWeapon) && switchToWeapon == 0 {
		for _, slot := range gladiator.Slots {
			if slot != 0 {
				switchToWeapon = slot
				break
			}
		}
	}

	// TODO: rewrite this
	if w.EntityExists(switchToWeapon) && gladiator.ActiveWeapon != switchToWeapon {
		// TODO: for weapon sway we would need to introduce another entity
		// (basically hands) which we would move around and actually use to
		// implement sway with.
		// TODO: make weapon switching predicted when we make CreateEntity work in speculative mode
		if !info.Speculating {
			gladiator.ActiveWeapon = 0

			if w.EntityExists(gladiator.ActiveWeaponFirstPersonProp) {
				w.Delete.Set(gladiator.ActiveWeaponFirstPersonProp, struct{}{})
			}
			gladiator.ActiveWeaponFirstPersonProp = 0

			// Now we can switch the weapons

			if weapon, ok := SceneGetEntity[Weapon](w, switchToWeapon); ok {
				gladiator.ActiveWeaponFirstPersonProp = weapon.CreateProp(w, info)
				w.SetParent(gladiator.ActiveWeaponFirstPersonProp, gladiator.FirstPersonHands)
				w.VisibilityMask.Set(gladiator.ActiveWeaponFirstPersonProp,
					VisibilityMask{Mask: 0b01, Camera: gladiator.FirstPersonCamera})

				gladiator.ActiveWeapon = switchToWeapon
			}
		}
	}

	// TODO: under some conditions we should autoselect a gun for the player

	if weapon, ok := SceneGetEntity[Weapon](w, gladiator.ActiveWeapon); ok {
		var buttons WeaponButtons
		if gladiator.HeldButtons&uint64(1<<ButtonAttack) != 0 {
			buttons |= WeaponTrigger
		}

		shootT := w.GetGlobalTransform(id).
			Mul(gmath.TRS3f64{
				T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
				R: gmath.Rot3InPlane(gmath.Vecf32{1, 0, 0}, gmath.Vec3f32{}, 2*math.Pi*gladiator.LookDir[0]).
					Mul(gmath.Rot3InPlane(planeYZ, 2*math.Pi*gladiator.LookDir[1])),
				S: gmath.Mat3x3UOne[float32](),
			}.Compose())

			/*
				ray := physics.Ray{
					Origin:    shootT.T,
					Direction: shootT.M.Mulv(gmath.Vec3f32{0, 1, 0}).Normalize(),
					TMax:      100,
				}
				var contactBuffer struct {
					MyHitBuffer[physics.RayHit]
				}
				w.physicsSystem.TraceRay(
					ray,
					func(body physics.BodyID) bool {
						if body == physics.BodyID(id.Index()) {
							return false
						}
						return true
					},
					&contactBuffer)
				for i, hit := range contactBuffer.MyHitBuffer {
					hitPos := ray.F(hit.Geometry.T)

					// _ = hitPos
					log.Println(i, hit.Geometry.T, hitPos)
				}
			*/

		shootv, _ := w.Velocity.Get(id)

		// TODO: shooter id we pass should be that of player, actually. Maybe we
		// should have a component to attribute kills and damage to something
		// else.
		stepResult := weapon.WeaponSubstep(w, gladiator.ActiveWeapon, []ecs.ID{gladiator.ActiveWeaponFirstPersonProp}, id, shootT, shootv, buttons, info)
		// TODO: apply some part of the recoil as viewpunch?
		// TODO: make sure we don't overflow LookDir
		gladiator.LookDir[0] += stepResult.Recoil[0]
		gladiator.LookDir[1] += stepResult.Recoil[1]
	}

	// TODO: factor this out?
	w.SetTransform(gladiator.FirstPersonCamera,
		gmath.TRS3f64{
			T: gmath.Vec3f64{0, 0, float64(gladiatorStats.StandingViewHeight)},
			R: gmath.Rot3InPlane(planeXY, 2*math.Pi*gladiator.LookDir[0]).
				Mul(gmath.Rot3InPlane(planeYZ, 2*math.Pi*gladiator.LookDir[1])),
			S: gmath.Mat3x3UOne[float32](),
		})
}

func (gladiator Gladiator) CharacterStep(w *Scene, id ecs.ID, info *UpdateParams) {
	// TODO: do not call this here?
	gladiator.CharacterSubstep(w, id, TimestampedInputCmd{}, info)

	// TODO: this is incredibly gross and ugly, FIXME
	gladiator = mustOk(SceneGetEntity[Gladiator](w, id))

	velocity, _ := w.Velocity.Get(id)

	// TODO: more elaborate viewmodel sway
	w.SetTransform(gladiator.FirstPersonHands,
		gmath.TRS3f64{
			T: gmath.Vec3f64{0, math.Sin(float64(w.Now)/1e9*6) * 0.03 * min(float64(velocity.Linear.Length()/6), 1), 0},
			R: gmath.Rot3One(),
			S: gmath.Mat3x3UOne[float32](),
		})

	trs := w.GetTransform(id)

	rotation := trs.R.Mul(gmath.Rot3InPlane(planeXY, 2*math.Pi*gladiator.LookDir[0]))

	move := gladiator.MoveVec
	if lengthSqr := move.Dot(move); lengthSqr > 1 {
		move = move.Scale(1 / float32(math.Sqrt(float64(lengthSqr))))
	}

	localVel := rotation.Inverse().Rotate(velocity.Linear)
	if gladiator.Supported {
		localVel[0] = move[0] * gladiatorStats.WalkVelocity
		localVel[1] = move[1] * gladiatorStats.WalkVelocity
		if gladiator.HeldButtons&(1<<ButtonJump) != 0 {
			localVel[2] = 4
		}
	}
	velocity.Linear = rotation.Rotate(localVel)

	if !gladiator.Supported {
		velocity.Linear = velocity.Linear.Add(w.Globals().Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
	}

	velocity.Linear = gladiator.asdasd(w, id, velocity.Linear, info.Δt)

	if gladiator.Supported {
		gladiator.Steps += float64(velocity.Linear.Length()) * durationToFloatSeconds(info.Δt)
	}
	if gladiator.Steps > 3 {
		w.SoundEffect.Set(id, SoundEmitter{
			Effect:      "step.wav",
			Attenuation: 1,
			PlayTime:    w.Now,
		})
		gladiator.Steps = 0
	}

	w.Entity.Set(id, gladiator)
	w.Velocity.Set(id, velocity)
}

func planeNormal(plane gmath.Vec4f32) gmath.Vec3f32 {
	return gmath.Vec3f32{plane[0], plane[1], plane[2]}
}

func planeSignedDistance(plane gmath.Vec4f32, point gmath.Vec3f32) float32 {
	return point.Dot(planeNormal(plane)) + plane[3]
}

// TODO: introduce typedefs to physics for query types

type gladiatorMovementQueryPipeline struct {
	player physics.BodyID
	hits   []physics.SceneQueryHit[physics.OverlapHit]
}

func (*gladiatorMovementQueryPipeline) FilterLayer(layer int) bool {
	// TOOD: we should just poke the rule table
	return layer == int(CollisionLayerNonMoving) || layer == int(CollisionLayerMoving)
}

func (a *gladiatorMovementQueryPipeline) FilterBody(body physics.BodyID) bool {
	return body != a.player
}

func (a *gladiatorMovementQueryPipeline) Hit(x physics.SceneQueryHit[physics.OverlapHit]) int {
	a.hits = append(a.hits, x)
	return 2
}

func (gladiator *Gladiator) asdasd(w *Scene, id ecs.ID, velocity gmath.Vec3f32, Δt time.Duration) gmath.Vec3f32 {
	trs := w.GetTransform(id)

	up := gmath.Vec3f32{0, 0, 1}

	var planes []gmath.Vec4f32

	wasSupported := gladiator.Supported

	gladiator.Supported = false

	contactBuffer := gladiatorMovementQueryPipeline{
		player: physics.BodyID(id.Index()),
	}
	w.physicsSystem.Overlap(
		physics.Overlap{
			Pos:                   trs.T,
			Rot:                   trs.R,
			Scale:                 gmath.Vec3Ones[float32](),
			Shape:                 getShape(w, id),
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

// TODO: should probs be a freestanding method
// TODO: allow it not to assume Gladiator? E.g. make things an interface.
func (scene *Scene) GiveWeapon(id ecs.ID, weapon ecs.ID) {
	char := mustOk(SceneGetEntity[Gladiator](scene, id))
	freeSlot := slices.Index(char.Slots[:], 0)
	char.Slots[freeSlot] = weapon
	scene.Entity.Set(id, char)

	scene.SetParent(weapon, id)
}
