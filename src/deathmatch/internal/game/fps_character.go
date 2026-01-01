package game

import (
	"math"
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
	"worldspawn/physics"
)

// TODO: name buttons and input commands in a way so that it's clear that they
// are player-specific. For vehicles we'd have a different set of input
// commands, probably even different between e.g. wheeled vehicles and planes.

type Button int8

const (
	_ Button = iota
	ButtonJump
	ButtonCrouch
	ButtonAttack
	ButtonReload
)

// TODO: use SNORM for movement velocity and look direction?

type (
	InputCmdDLookX        float32
	InputCmdDLookY        float32
	InputCmdMoveX         float32
	InputCmdMoveY         float32
	InputCmdPressButton   Button
	InputCmdReleaseButton Button
)

type Slot int8

// TODO: I guess we should make character controller also be an interface?
// Though maybe not, this seems less clear cut compared to weapons.

// TODO: we need a new way to specify what stuff to draw as viewmodels

// TODO: shove a viewmodel offset into fpsCharacter?

// TODO: this code is in serious need of work!!!

// TODO: call this "Player"? Or idk.
type FPSCharacter struct {
	// TODO: move this into a separate component?
	Camera ecs.ID

	// TODO: shape
	StandingHeight float32

	WalkVelocity                float32
	BackwardsWalkVelocityFactor float32
	WalkAcceleration            float32
	AirAcceleration             float32
	CosMaxSlopeAngle            float32
	MaxStepHeight               float32
	JumpVelocity                float32

	StandingViewHeight float32

	// TODO: give these more descriptive names
	Look    geometry.Vec2
	Move    geometry.Vec2
	Buttons uint64

	Supported bool

	ActiveWeapon           ecs.ID
	ActiveWeaponViewmodel  ecs.ID
	ActiveWeaponWorldmodel ecs.ID
	Weapons                []ecs.ID
}

var _ Controllable = FPSCharacter{}

func (entity FPSCharacter) ControllableUpdateSubtick(w *Scene, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	inventory, _ := w.ArmedCharacter.Get(id)

	var switchToWeapon ecs.ID

	switch cmd := cmd.Cmd.(type) {
	case InputCmdDLookX:
		entity.Look[0] += float32(cmd)
	case InputCmdDLookY:
		entity.Look[1] += float32(cmd)
	case InputCmdMoveX:
		entity.Move[0] = float32(cmd)
	case InputCmdMoveY:
		entity.Move[1] = float32(cmd)
	case InputCmdPressButton:
		entity.Buttons |= uint64(1) << cmd
	case InputCmdReleaseButton:
		entity.Buttons &^= uint64(1) << cmd
	case Slot:
		if 0 <= int(cmd) && int(cmd) < len(inventory.Slots) {
			switchToWeapon = inventory.Slots[cmd]
		}
	default:
		// Right now we can get nil here when ControllableUpdateSubtick is
		// called from ControllableUpdate but ugh.
		// panic("unreachable")
	}

	entity.Look[0] = float32(math.Mod(float64(entity.Look[0]), 1))
	entity.Look[1] = min(max(entity.Look[1], -0.25), 0.25)

	if !w.IsEntityValid(entity.ActiveWeapon) && len(inventory.Slots) > 0 {
		switchToWeapon = inventory.Slots[0]
	}

	if w.IsEntityValid(switchToWeapon) {
		// TODO: don't delete the view and worldmodel entities but just hide
		// them?
		if w.IsEntityValid(entity.ActiveWeaponViewmodel) {
			w.Delete.Set(entity.ActiveWeaponViewmodel, struct{}{})
		}
		entity.ActiveWeaponViewmodel = 0

		// Now we can switch the weapons

		entity.ActiveWeapon = switchToWeapon

		if !info.Speculating {
			if weapon, ok := assertEntity[Weapon](w, entity.ActiveWeapon); ok {
				entity.ActiveWeaponViewmodel = weapon.WeaponCreateGeometry(w, info)

				w.Viewmodel2.Set(entity.ActiveWeaponViewmodel,
					Viewmodel2{
						Camera: entity.Camera,
						Mode:   1,
					})
				w.ParentTo(entity.ActiveWeaponViewmodel, entity.Camera)
			}
		}
	}

	if weapon, ok := assertEntity[Weapon](w, entity.ActiveWeapon); ok {
		var buttons WeaponButtons
		if entity.Buttons&uint64(1<<ButtonAttack) != 0 {
			buttons |= WeaponTrigger
		}

		shootpos, _ := w.GetGlobalTRS(id)
		shootpos = shootpos.Mul(geometry.DTRS3{
			T: geometry.DVec3{0, 0, float64(entity.StandingViewHeight)},
			R: geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*entity.Look[0]).Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*entity.Look[1])),
			S: geometry.Vec3Broadcast(1),
		})

		updateVisual := weapon.WeaponUpdateSubtick(w, entity.ActiveWeapon, shootpos, buttons, info)
		if updateVisual != nil {
			updateVisual(w, entity.ActiveWeaponViewmodel)
		}
	}

	// TODO: avoid unnecessary updates
	w.Entity.Set(id, entity)

	// TODO: factor this out
	w.SetLocalTRS(entity.Camera, geometry.DTRS3{
		T: geometry.DVec3{0, 0, float64(entity.StandingViewHeight)},
		R: geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*entity.Look[0]).Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*entity.Look[1])),
		S: geometry.Vec3Broadcast(1),
	})
}

func (entity FPSCharacter) ControllableUpdate(w *Scene, id ecs.ID, info *UpdateParams) {
	// TODO: fix this garbage
	entity.ControllableUpdateSubtick(w, id, TimestampedInputCmd{}, info)
}

var _ UpdateBeforePhysics = FPSCharacter{}

func (entity FPSCharacter) UpdateBeforePhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	trs, _ := w.GetGlobalTRS(id)
	velocity, _ := w.Velocity.Get(id)

	rotation := trs.R.Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*entity.Look[0]))

	move := entity.Move
	if lenSq := move.LengthSq(); lenSq > 1 {
		move = move.Scale(1 / float32(math.Sqrt(float64(lenSq))))
	}

	localVel := rotation.Inverse().Rotate(velocity.Linear)
	if entity.Supported {
		localVel[0] = move[0] * entity.WalkVelocity
		localVel[1] = move[1] * entity.WalkVelocity
		if entity.Buttons&(1<<ButtonJump) != 0 {
			localVel[2] = 4
		}
	}
	velocity.Linear = rotation.Rotate(localVel)

	if !entity.Supported {
		velocity.Linear = velocity.Linear.Add(w.Globals().Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
	}

	velocity.Linear = entity.asdasd(w, id, velocity.Linear, info.Δt)

	w.Entity.Set(id, entity)
	w.Velocity.Set(id, velocity)
}

// TODO: rename to Inventory or something else
type ArmedCharacter struct {
	Slots []ecs.ID
}

func planeNormal(plane geometry.Vec4) geometry.Vec3 {
	return geometry.Vec3{plane[0], plane[1], plane[2]}
}

func planeSignedDistance(plane geometry.Vec4, point geometry.Vec3) float32 {
	return point.Dot(planeNormal(plane)) + plane[3]
}

func (entity *FPSCharacter) asdasd(w *Scene, id ecs.ID, velocity geometry.Vec3, Δt time.Duration) geometry.Vec3 {
	trs, _ := w.GetGlobalTRS(id)

	up := geometry.Vec3{0, 0, 1}

	var planes []geometry.Vec4

	hits := make([]physics.QueryHit, 100)
	n := w.physicsSystem.QueryShape(
		getShape(w, id),
		trs.T,
		trs.R,
		trs.S,
		velocity.NormalizedOr(geometry.Vec3{}),
		0.1,
		physics.QueryFilter{Ignore: physics.BodyID(id)},
		hits)
	hits = hits[:n]

	entity.Supported = false

	for _, contact := range hits {
		normal := contact.Normal.Scale(-1)
		if normal.Dot(up) < 0.7 {
			if false {
				// This prevents us from walking up steep ramps, but has an issue in
				// that sometimes we get a ghost steep ramp
				normal2 := normal.Cross(up).NormalizedOr(geometry.Vec3{}).Cross(up).Scale(-1)
				planes = append(planes, geometry.Vec4{
					normal2[0],
					normal2[1],
					normal2[2],
					-contact.Depth + 0.1,
				})
			}
		} else {
			entity.Supported = true
		}
		planes = append(planes, geometry.Vec4{
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

	if velocity.LengthSq() < minSpeed*minSpeed {
		velocity = geometry.Vec3{}
	}

	/*
		chit := w.physicsSystem.QuerySweptShapeClosestHit(
			getShape(shape),
			positionRotation.Position,
			positionRotation.Rotation,
			geometry.Vec3Broadcast(1),
			velocity.Linear.Scale(float32(durationToFloatSeconds(Δt))),
			physics.QueryFilter{Ignore: physics.BodyID(id)})
	*/

	return velocity
}
