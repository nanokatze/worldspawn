package game

import (
	"reflect"
)

// TODO: give these a type
// TODO: rename to axis?
// TODO: these depend on action sets actually
// TODO: move action stuff into a subpackage? This is mostly just a glue between
// InputCmds and user stuff.
const (
	_ = iota

	ActionJump
	ActionCrouch
	ActionAttack
	ActionReload

	ActionSlot0
	ActionSlot1
	ActionSlot2
	ActionSlot3

	ActionMoveX
	ActionMoveY

	ActionDLookX
	ActionDLookY
)

// TODO: we need a tracker object so that we can normalize value per action and
// filter things out
func AppendAction(dst []TimestampedInputCmd, time Time, action int, value float32) []TimestampedInputCmd {
	cmd := actionToInputCmd(action, value)
	if cmd != nil {
		dst = append(dst, TimestampedInputCmd{Time: time, Cmd: cmd})
	}
	return dst
}

// TODO: with some extra effort we can make InputCmd values private
func actionToInputCmd(action int, value float32) InputCmd {
	switch action {
	case ActionJump, ActionCrouch, ActionAttack, ActionReload:
		if value != 0 {
			return ButtonDown(action)
		} else {
			return ButtonUp(action)
		}

	case ActionSlot0, ActionSlot1, ActionSlot2, ActionSlot3:
		// TODO: we should do nothing if value == 0
		return Slot(action - ActionSlot0)

	case ActionMoveX:
		return MoveX(value)

	case ActionMoveY:
		return MoveY(value)

	case ActionDLookX:
		return DLookX(value)

	case ActionDLookY:
		return DLookY(value)

	default:
		panic("unknown action")
	}
}

// TODO: serialize input cmds to []byte immediately, rather than pass structs
// around?

type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

type InputCmd any

// TODO: use SNORM for move and look?

// TODO: prefix these with InputCmd probably

type DLookX float32
type DLookY float32

type MoveX float32
type MoveY float32

// TODO: replace generic ButtonDown and ButtonUp with a definition per each
// button and a
type Button uint8

const (
	_ Button = iota
	ButtonJump
	ButtonCrouch
	ButtonAttack
	ButtonReload
)

type ButtonDown Button
type ButtonUp Button

// TODO: rename this into something more descriptive (e.g. UseWeaponInSlot).
// Also, we actually want to use weapon by its ID, and slots should be
// user-configurable probably. Actually, this would prevent user from having
// multiple instances of the same weapon, so we need to rethink that. I guess we
// should call these things "slots" then? TODO: think harder
//
// TODO: remove in favor of slot buttons, so that weapon switching is entirely
// dictated by the game.
type Slot int8

var InputCmds = []reflect.Type{
	reflect.TypeFor[DLookX](),
	reflect.TypeFor[DLookY](),
	reflect.TypeFor[MoveX](),
	reflect.TypeFor[MoveY](),
	reflect.TypeFor[ButtonDown](),
	reflect.TypeFor[ButtonUp](),
	reflect.TypeFor[Slot](),
}
