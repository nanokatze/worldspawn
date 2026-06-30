package game

import (
	"reflect"
)

// TODO: rename this file

type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

// TODO: replace most input commands with a simple axis

// TODO: we need two types of InputCmds: passthrough to Character, and
// administrative things like Respawn, etc.
type InputCmd any

// TODO: kill Button?
type Button int8

const (
	_ Button = iota
	ButtonJump
	ButtonCrouch
	ButtonAttack
	ButtonReload
	ButtonDash
)

// TODO: use SNORM for movement velocity and look direction?

type (
	InputCmdDLookXY       float32
	InputCmdDLookYZ       float32
	InputCmdMoveX         float32
	InputCmdMoveY         float32
	InputCmdPressButton   Button
	InputCmdReleaseButton Button
)

// TODO: replace with buttons?
type Slot int8

var InputCmdTypes = []reflect.Type{
	reflect.TypeFor[InputCmdDLookXY](),
	reflect.TypeFor[InputCmdDLookYZ](),
	reflect.TypeFor[InputCmdMoveX](),
	reflect.TypeFor[InputCmdMoveY](),
	reflect.TypeFor[InputCmdPressButton](),
	reflect.TypeFor[InputCmdReleaseButton](),
	reflect.TypeFor[Slot](),
}
