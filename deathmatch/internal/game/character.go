package game

import (
	"math"
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/physics"
)

var playerStats = struct {
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

// TODO: turn this into an interface?
type Character struct {
	// FirstPersonCamera is always a descendant of the Character,
	FirstPersonCamera ecs.ID

	// TODO: give these more descriptive names
	Look    gmath.Vec2f32
	Move    gmath.Vec2f32
	Buttons uint64

	Supported bool

	// This entity is what the viewmodel gets parented to.
	Hands ecs.ID

	ActiveWeapon           ecs.ID
	ActiveWeaponViewmodel  ecs.ID
	ActiveWeaponWorldmodel ecs.ID

	Slots [4]ecs.ID
}

func (Character) entity() {}

// TODO: this should accept its own input things.
func (char Character) CharacterSubstep(w *Scene, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	var switchToWeapon ecs.ID
	switch cmd := cmd.Cmd.(type) {
	case InputCmdDLookX:
		char.Look[0] = float32(math.Mod(float64(char.Look[0]+float32(cmd)), 1))
	case InputCmdDLookY:
		char.Look[1] = min(max(char.Look[1]+float32(cmd), -0.25), 0.25)
	case InputCmdMoveX:
		char.Move[0] = float32(cmd)
	case InputCmdMoveY:
		char.Move[1] = float32(cmd)
	case InputCmdPressButton:
		char.Buttons |= uint64(1) << cmd
	case InputCmdReleaseButton:
		char.Buttons &^= uint64(1) << cmd
	case Slot:
		if !(0 <= int(cmd) && int(cmd) < len(char.Slots)) {
			break
		}

		switchToWeapon = char.Slots[cmd]

	default:
		// TODO: we should not hit this with nil either
		if cmd != nil {
			panic("unreachable")
		}
	}

	if !w.IsEntityValid(char.ActiveWeapon) && switchToWeapon == 0 {
		for _, slot := range char.Slots {
			if slot != 0 {
				switchToWeapon = slot
				break
			}
		}
	}

	// TODO: rewrite this
	if w.IsEntityValid(switchToWeapon) && char.ActiveWeapon != switchToWeapon {
		// TODO: for weapon sway we would need to introduce another entity
		// (basically hands) which we would move around and actually use to
		// implement sway with.
		// TODO: make weapon switching predicted when we make CreateEntity work in speculative mode
		if !info.Speculating {
			char.ActiveWeapon = 0

			if w.IsEntityValid(char.ActiveWeaponViewmodel) {
				w.Delete.Set(char.ActiveWeaponViewmodel, struct{}{})
			}
			char.ActiveWeaponViewmodel = 0

			// Now we can switch the weapons

			if weapon, ok := SceneGetEntity[Weapon](w, switchToWeapon); ok {
				char.ActiveWeaponViewmodel = weapon.WeaponCreateGeometry(w, char.Hands, info)

				w.VisibilityMask.Set(char.ActiveWeaponViewmodel, VisibilityMask{Mask: 0b01, Camera: char.FirstPersonCamera})

				char.ActiveWeapon = switchToWeapon
			}
		}
	}

	// TODO: under some conditions we should autoselect a gun for the player

	if weapon, ok := SceneGetEntity[Weapon](w, char.ActiveWeapon); ok {
		var buttons WeaponButtons
		if char.Buttons&uint64(1<<ButtonAttack) != 0 {
			buttons |= WeaponTrigger
		}

		shootpos, _ := w.GetGlobalTransform(id)
		shootpos = shootpos.Mul(gmath.TRS3f64{
			T: gmath.Vec3f64{0, 0, float64(playerStats.StandingViewHeight)},
			R: gmath.Rot3InPlane(gmath.Vec3f32{0, 0, -1}, 2*math.Pi*char.Look[0]).Mul(gmath.Rot3InPlane(gmath.Vec3f32{-1, 0, 0}, 2*math.Pi*char.Look[1])),
			S: gmath.Shcale3One(),
		}.ToAffine())

		updateVisual := weapon.WeaponSubstep(w, char.ActiveWeapon, id, shootpos, buttons, info)
		if updateVisual != nil {
			updateVisual(w, char.ActiveWeaponViewmodel)
		}
	}

	// TODO: avoid unnecessary updates
	w.Entity.Set(id, char)

	// TODO: factor this out
	w.Transform.Set(char.FirstPersonCamera, gmath.TRS3f64{
		T: gmath.Vec3f64{0, 0, float64(playerStats.StandingViewHeight)},
		R: gmath.Rot3InPlane(gmath.Vec3f32{0, 0, -1}, 2*math.Pi*char.Look[0]).Mul(gmath.Rot3InPlane(gmath.Vec3f32{-1, 0, 0}, 2*math.Pi*char.Look[1])),
		S: gmath.Shcale3One(),
	}.ToAffine())
}

func (char Character) CharacterUpdate(w *Scene, id ecs.ID, info *UpdateParams) {
	// TODO: do not call this here, properly move stuff from PlayerSubstep to here
	char.CharacterSubstep(w, id, TimestampedInputCmd{}, info)

	velocity, _ := w.Velocity.Get(id)

	// TODO: more elaborate viewmodel sway
	w.Transform.Set(char.Hands, gmath.TRS3f64{
		T: gmath.Vec3f64{0, math.Sin(float64(w.Now)/1e9*6) * 0.03 * min(float64(velocity.Linear.Length()/6), 1), 0},
		R: gmath.Rot3One(),
		S: gmath.Shcale3One(),
	}.ToAffine())
}

var _ UpdateBeforePhysics = Character{}

func (char Character) UpdateBeforePhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	transform, _ := w.GetGlobalTransform(id)
	velocity, _ := w.Velocity.Get(id)

	trs := gmath.TRS3FromAffine(transform)

	rotation := trs.R.Mul(gmath.Rot3InPlane(gmath.Vec3f32{0, 0, -1}, 2*math.Pi*char.Look[0]))

	move := char.Move
	if lenSq := move.Dot(move); lenSq > 1 {
		move = move.Scale(1 / float32(math.Sqrt(float64(lenSq))))
	}

	localVel := rotation.Inverse().Rotate(velocity.Linear)
	if char.Supported {
		localVel[0] = move[0] * playerStats.WalkVelocity
		localVel[1] = move[1] * playerStats.WalkVelocity
		if char.Buttons&(1<<ButtonJump) != 0 {
			localVel[2] = 4
		}
	}
	velocity.Linear = rotation.Rotate(localVel)

	if !char.Supported {
		velocity.Linear = velocity.Linear.Add(w.Globals().Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
	}

	velocity.Linear = char.asdasd(w, id, velocity.Linear, info.Δt)

	w.Entity.Set(id, char)
	w.Velocity.Set(id, velocity)
}

func planeNormal(plane gmath.Vec4f32) gmath.Vec3f32 {
	return gmath.Vec3f32{plane[0], plane[1], plane[2]}
}

func planeSignedDistance(plane gmath.Vec4f32, point gmath.Vec3f32) float32 {
	return point.Dot(planeNormal(plane)) + plane[3]
}

func (char *Character) asdasd(w *Scene, id ecs.ID, velocity gmath.Vec3f32, Δt time.Duration) gmath.Vec3f32 {
	transform, _ := w.GetGlobalTransform(id)

	trs := gmath.TRS3FromAffine(transform)

	up := gmath.Vec3f32{0, 0, 1}

	var planes []gmath.Vec4f32

	hits := make([]physics.QueryHit, 100)
	n := w.physicsSystem.QueryShape(
		getShape(w, id),
		trs.T,
		trs.R,
		gmath.Vec3Ones[float32](),
		velocity.NormalizeOr(gmath.Vec3f32{}),
		0.1,
		physics.QueryFilter{Ignore: physics.BodyID(id)},
		hits)
	hits = hits[:n]

	char.Supported = false

	for _, contact := range hits {
		normal := contact.Normal.Scale(-1)
		if normal.Dot(up) < 0.7 {
			if false {
				// This prevents us from walking up steep ramps, but has an issue in
				// that sometimes we get a ghost steep ramp
				normal2 := normal.Cross(up).NormalizeOr(gmath.Vec3f32{}).Cross(up).Scale(-1)
				planes = append(planes, gmath.Vec4f32{
					normal2[0],
					normal2[1],
					normal2[2],
					-contact.Depth + 0.1,
				})
			}
		} else {
			char.Supported = true
		}
		planes = append(planes, gmath.Vec4f32{
			normal[0],
			normal[1],
			normal[2],
			-contact.Depth + 0.1,
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

	/*
		chit := w.physicsSystem.QuerySweptShapeClosestHit(
			getShape(shape),
			positionRotation.Position,
			positionRotation.Rotation,
			geometry.Vec3Ones(),
			velocity.Linear.Scale(float32(durationToFloatSeconds(Δt))),
			physics.QueryFilter{Ignore: physics.BodyID(id)})
	*/

	return velocity
}
