package game

import (
	"reflect"
)

type TimestampedInputCmd struct {
	Time Time
	Cmd  InputCmd
}

type InputCmd any

type (
	InputCmdDLookXY      float32
	InputCmdDLookYZ      float32
	InputCmdMoveX        float32
	InputCmdMoveY        float32
	InputCmdJump         bool
	InputCmdCrouch       bool
	InputCmdAttack       bool
	InputCmdReload       bool
	InputCmdDash         bool
	InputCmdSwitchWeapon int8
)

var InputCmdTypes = []reflect.Type{
	reflect.TypeFor[InputCmdDLookXY](),
	reflect.TypeFor[InputCmdDLookYZ](),
	reflect.TypeFor[InputCmdMoveX](),
	reflect.TypeFor[InputCmdMoveY](),
	reflect.TypeFor[InputCmdJump](),
	reflect.TypeFor[InputCmdCrouch](),
	reflect.TypeFor[InputCmdAttack](),
	reflect.TypeFor[InputCmdReload](),
	reflect.TypeFor[InputCmdDash](),
	reflect.TypeFor[InputCmdSwitchWeapon](),
}
