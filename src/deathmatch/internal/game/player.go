package game

import (
	"math"
	"reflect"
	"time"
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
	"worldspawn/physics"
)

// TODO: replace with []byte and do de/serialization at HandleInput time?
type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

type InputCmd any

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

// TODO: replace with buttons?
type Slot int8

var InputCmdTypes = []reflect.Type{
	reflect.TypeFor[InputCmdDLookX](),
	reflect.TypeFor[InputCmdDLookY](),
	reflect.TypeFor[InputCmdMoveX](),
	reflect.TypeFor[InputCmdMoveY](),
	reflect.TypeFor[InputCmdPressButton](),
	reflect.TypeFor[InputCmdReleaseButton](),
	reflect.TypeFor[Slot](),
}

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

// TODO: split into an entity representing the actual player, and the character
// that can walk and stuff?
type Player struct {
	// TODO: move this into a separate component?
	Camera ecs.ID

	// TODO: give these more descriptive names
	Look    geometry.Vec2
	Move    geometry.Vec2
	Buttons uint64

	Supported bool

	// This entity is what the viewmodel gets parented to.
	Hands ecs.ID

	ActiveWeapon           ecs.ID
	ActiveWeaponViewmodel  ecs.ID
	ActiveWeaponWorldmodel ecs.ID

	Slots [4]ecs.ID
}

func (Player) entity() {}

func (player Player) PlayerSubstep(w *Scene, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	switch cmd := cmd.Cmd.(type) {
	case InputCmdDLookX:
		player.Look[0] = float32(math.Mod(float64(player.Look[0]+float32(cmd)), 1))
	case InputCmdDLookY:
		player.Look[1] = min(max(player.Look[1]+float32(cmd), -0.25), 0.25)
	case InputCmdMoveX:
		player.Move[0] = float32(cmd)
	case InputCmdMoveY:
		player.Move[1] = float32(cmd)
	case InputCmdPressButton:
		player.Buttons |= uint64(1) << cmd
	case InputCmdReleaseButton:
		player.Buttons &^= uint64(1) << cmd
	case Slot:
		if !(0 <= int(cmd) && int(cmd) < len(player.Slots)) {
			break
		}

		switchToWeapon := player.Slots[cmd]

		// TODO: rewrite this
		if player.ActiveWeapon != switchToWeapon {
			// TODO: for weapon sway we would need to introduce another entity
			// (basically hands) which we would move around and actually use to
			// implement sway with.
			// TODO: make weapon switching predicted when we make CreateEntity work in speculative mode
			if !info.Speculating {
				player.ActiveWeapon = 0

				// TODO: don't delete the view and worldmodel entities but just hide
				// them?
				if w.IsEntityValid(player.ActiveWeaponViewmodel) {
					w.Delete.Set(player.ActiveWeaponViewmodel, struct{}{})
				}
				player.ActiveWeaponViewmodel = 0

				// Now we can switch the weapons

				player.ActiveWeapon = switchToWeapon

				if weapon, ok := SceneGetEntity[Weapon](w, switchToWeapon); ok {
					player.ActiveWeaponViewmodel = weapon.WeaponCreateGeometry(w, player.Hands, info)

					w.Visibility.Set(player.ActiveWeaponViewmodel, Visibility{Mode: 1, Camera: player.Camera})
				}
			}
		}

	default:
		// TODO: we should not hit this with nil either
		if cmd != nil {
			panic("unreachable")
		}
	}

	// TODO: under some conditions we should autoselect a gun for the player

	if weapon, ok := SceneGetEntity[Weapon](w, player.ActiveWeapon); ok {
		var buttons WeaponButtons
		if player.Buttons&uint64(1<<ButtonAttack) != 0 {
			buttons |= WeaponTrigger
		}

		shootpos, _ := w.GetGlobalTRS(id)
		shootpos = shootpos.Mul(geometry.DTRS3{
			T: geometry.DVec3{0, 0, float64(playerStats.StandingViewHeight)},
			R: geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*player.Look[0]).Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*player.Look[1])),
			S: geometry.Vec3Broadcast(1),
		})

		updateVisual := weapon.WeaponSubstep(w, player.ActiveWeapon, id, shootpos, buttons, info)
		if updateVisual != nil {
			updateVisual(w, player.ActiveWeaponViewmodel)
		}
	}

	// TODO: avoid unnecessary updates
	w.Entity.Set(id, player)

	// TODO: factor this out
	w.SetLocalTRS(player.Camera, geometry.DTRS3{
		T: geometry.DVec3{0, 0, float64(playerStats.StandingViewHeight)},
		R: geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*player.Look[0]).Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*player.Look[1])),
		S: geometry.Vec3Broadcast(1),
	})
}

func (player Player) PlayerUpdate(w *Scene, id ecs.ID, info *UpdateParams) {
	// TODO: do not call this here, properly move stuff from PlayerSubstep to here
	player.PlayerSubstep(w, id, TimestampedInputCmd{}, info)

	velocity, _ := w.Velocity.Get(id)

	// TODO: more elaborate viewmodel sway
	w.SetLocalTRS(player.Hands, geometry.DTRS3{
		T: geometry.DVec3{0, math.Sin(float64(w.Now)/1e9*6) * 0.03 * min(float64(velocity.Linear.Length()/6), 1), 0},
		R: geometry.Rot3One(),
		S: geometry.Vec3Broadcast(1),
	})
}

var _ UpdateBeforePhysics = Player{}

func (player Player) UpdateBeforePhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	trs, _ := w.GetGlobalTRS(id)
	velocity, _ := w.Velocity.Get(id)

	rotation := trs.R.Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*player.Look[0]))

	move := player.Move
	if lenSq := move.LengthSq(); lenSq > 1 {
		move = move.Scale(1 / float32(math.Sqrt(float64(lenSq))))
	}

	localVel := rotation.Inverse().Rotate(velocity.Linear)
	if player.Supported {
		localVel[0] = move[0] * playerStats.WalkVelocity
		localVel[1] = move[1] * playerStats.WalkVelocity
		if player.Buttons&(1<<ButtonJump) != 0 {
			localVel[2] = 4
		}
	}
	velocity.Linear = rotation.Rotate(localVel)

	if !player.Supported {
		velocity.Linear = velocity.Linear.Add(w.Globals().Gravity.Scale(float32(durationToFloatSeconds(info.Δt))))
	}

	velocity.Linear = player.asdasd(w, id, velocity.Linear, info.Δt)

	w.Entity.Set(id, player)
	w.Velocity.Set(id, velocity)
}

func planeNormal(plane geometry.Vec4) geometry.Vec3 {
	return geometry.Vec3{plane[0], plane[1], plane[2]}
}

func planeSignedDistance(plane geometry.Vec4, point geometry.Vec3) float32 {
	return point.Dot(planeNormal(plane)) + plane[3]
}

func (player *Player) asdasd(w *Scene, id ecs.ID, velocity geometry.Vec3, Δt time.Duration) geometry.Vec3 {
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

	player.Supported = false

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
			player.Supported = true
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
