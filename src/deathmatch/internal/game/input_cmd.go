package game

import "reflect"

// TODO: replace with []byte and do de/serialization at HandleInput time?
type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

type InputCmd any

var InputCmdTypes = []reflect.Type{
	reflect.TypeFor[InputCmdDLookX](),
	reflect.TypeFor[InputCmdDLookY](),
	reflect.TypeFor[InputCmdMoveX](),
	reflect.TypeFor[InputCmdMoveY](),
	reflect.TypeFor[InputCmdPressButton](),
	reflect.TypeFor[InputCmdReleaseButton](),
	reflect.TypeFor[Slot](),
}
