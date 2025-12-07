package game

import "reflect"

type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

type InputCmd any

// TODO: use SNORM for movement velocity and look direction?

// TODO: naming
type (
	InputCmdDLookX float32
	InputCmdDLookY float32
)

type (
	InputCmdSetMovementVelocityX float32
	InputCmdSetMovementVelocityY float32
)

type Button int8

const (
	_ Button = iota
	ButtonJump
	ButtonCrouch
	ButtonAttack
	ButtonReload
)

type (
	InputCmdPressButton   Button
	InputCmdReleaseButton Button
)

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
	reflect.TypeFor[InputCmdDLookX](),
	reflect.TypeFor[InputCmdDLookY](),
	reflect.TypeFor[InputCmdSetMovementVelocityX](),
	reflect.TypeFor[InputCmdSetMovementVelocityY](),
	reflect.TypeFor[InputCmdPressButton](),
	reflect.TypeFor[InputCmdReleaseButton](),
	reflect.TypeFor[Slot](),
}
