package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

// TODO: replace with []byte and do de/serialization at HandleInput time?
type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

// TODO: we need two types of InputCmds: passthrough to Character, and
// administrative things like Respawn, etc.
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

type PlayableCharacter interface {
	Entity

	// TODO: rename back to Tick or Update tbh
	CharacterStep(w *Scene, id ecs.ID, info *UpdateParams)
	CharacterSubstep(w *Scene, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams)
}
