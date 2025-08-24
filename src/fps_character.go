package worldspawn

import (
	"math"
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
	"worldspawn/physics"
)

// TODO: I guess we should make character controller also be an interface?
// Though maybe not, this seems less clear cut compared to weapons.

// TODO: we need a new way to specify what stuff to draw as viewmodels

// TODO: shove a viewmodel offset into fpsCharacter?

// TODO: this code is in serious need of work!!!

type FPSCharacter struct {
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

	ActiveWeapon ecs.ID
	Supported    bool
}

func init() {
	registerEntity[FPSCharacter]()
}

var _ Character = FPSCharacter{}

func (entity FPSCharacter) CharacterUpdate(w *World, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	positionRotation, _ := w.TranslationRotation.Load(id)
	inventory, _ := w.ArmedCharacter.Load(id)

	// should this be subtick time?
	now := w.Now

	oldActiveWeapon := entity.ActiveWeapon

	switch cmd := cmd.Cmd.(type) {
	case DLookX:
		entity.Look.X = float32(math.Mod(float64(entity.Look.X+float32(cmd)), 1))
	case DLookY:
		entity.Look.Y = min(max(entity.Look.Y+float32(cmd), -0.25), 0.25)
	case MoveX:
		entity.Move.X = float32(cmd)
	case MoveY:
		entity.Move.Y = float32(cmd)
	case ButtonDown:
		entity.Buttons |= uint64(1) << cmd
	case ButtonUp:
		entity.Buttons &^= uint64(1) << cmd
	case Slot:
		if 0 <= int(cmd) && int(cmd) < len(inventory.Slots) {
			entity.ActiveWeapon = inventory.Slots[cmd]
		}
	default:
		// TODO: optionally print this command for debugging and stuff
	}

	// TODO: instead of checking for zero, we should check for validity
	if !w.IsEntityValid(entity.ActiveWeapon) && len(inventory.Slots) > 0 {
		entity.ActiveWeapon = inventory.Slots[0]
	}

	if oldActiveWeapon != entity.ActiveWeapon {
		weapon, ok := assertEntity[WeaponDeployedInterface](w, entity.ActiveWeapon)
		if ok {
			weapon.WeaponDeployed(w, entity.ActiveWeapon, id, now, info.Δt)
		}
	}

	if w.IsEntityValid(entity.ActiveWeapon) {
		// TODO: we should add a component on the player which will specify offset
		// wrt view for some weapons. Weapons will need to carry some sort of
		// identifier for this purpose.
		elevation := float32(0)
		// HACK: we should probably just let the weapon specify elevation or provide an ID
		/*
			if fpsCharacter.ActiveWeapon != inventory.Slots[1] {
				elevation = 0
			}
		*/

		w.WeaponAim.Store(entity.ActiveWeapon, WeaponAim{
			ShootPos: positionRotation.Translation.
				Add(geometry.DVec3FromVec3(positionRotation.Rotation.Rotate(geometry.Vec3{0, 0, entity.StandingViewHeight}))),
			ShootRotation: positionRotation.Rotation.
				Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*entity.Look.X)).
				Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*entity.Look.Y-elevation)),
			Buttons: entity.Buttons,
		})

		if weapon, ok := assertEntity[Weapon](w, entity.ActiveWeapon); ok {
			recoil := weapon.WeaponUpdateSubtick(w, entity.ActiveWeapon, id, now, info)

			if recoil.LengthSq() > 0 {
				viewPunch, ok := w.ViewPunch.Load(id)
				if !ok {
					viewPunch = geometry.Rot3One()
				}
				viewPunch = viewPunch.Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, -recoil[0]))
				w.ViewPunch.Store(id, viewPunch)
			}
		}
	}

	// TODO: avoid unnecessary updates
	w.Entity.Store(id, entity)
}

var _ UpdateBeforePhysics = FPSCharacter{}

func (fpsCharacter FPSCharacter) UpdateBeforePhysics(w *World, id ecs.ID, info *UpdateParams) {
	positionRotation, _ := w.TranslationRotation.Load(id)
	velocity, _ := w.Velocity.Load(id)

	rotation := positionRotation.Rotation.
		Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*fpsCharacter.Look.X))

	move := fpsCharacter.Move
	if lenSq := move.LengthSq(); lenSq > 1 {
		move = move.Scale(1 / float32(math.Sqrt(float64(lenSq))))
	}

	localVel := rotation.Inverse().Rotate(velocity.Linear)
	if fpsCharacter.Supported {
		localVel[0] = move.X * fpsCharacter.WalkVelocity
		localVel[1] = move.Y * fpsCharacter.WalkVelocity
		if fpsCharacter.Buttons&(1<<ButtonJump) != 0 {
			localVel[2] = 4
		}
	}
	velocity.Linear = rotation.Rotate(localVel)

	if !fpsCharacter.Supported {
		velocity.Linear = velocity.Linear.Add(w.Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
	}

	velocity.Linear = fpsCharacter.asdasd(w, id, velocity.Linear, info.Δt)

	w.Entity.Store(id, fpsCharacter)
	w.Velocity.Store(id, velocity)
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

func (fpsCharacter *FPSCharacter) asdasd(w *World, id ecs.ID, velocity geometry.Vec3, Δt time.Duration) geometry.Vec3 {
	positionRotation, _ := w.TranslationRotation.Load(id)
	shape, _ := w.CollisionGeometry.Load(id)

	up := geometry.Vec3{0, 0, 1}

	var planes []geometry.Vec4

	hits := make([]physics.QueryHit, 100)
	n := w.physicsSystem.QueryShape(
		getShape(shape),
		positionRotation.Translation,
		positionRotation.Rotation,
		geometry.Vec3Broadcast(1),
		velocity.NormalizedOr(geometry.Vec3{}),
		0.1,
		physics.QueryFilter{Ignore: physics.BodyID(id)},
		hits)
	hits = hits[:n]

	fpsCharacter.Supported = false

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
			fpsCharacter.Supported = true
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
