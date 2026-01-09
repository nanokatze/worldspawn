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

type Player struct {
	Score struct {
		Kills  int32
		Deaths int32
	}

	Character ecs.ID
}

func (Player) entity() {}

// TODO: delete, client should figure this out by itself
func (player Player) Camera(w *Scene) ecs.ID {
	if player.Character == 0 {
		return 0 // TODO: get rid of this and make the callers care
	}
	char := mustOk(SceneGetEntity[Character](w, player.Character))
	return char.FirstPersonCamera
}

func (w *Scene) HandleInput(playerID ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	player := mustOk(SceneGetEntity[Player](w, playerID))
	char := mustOk(SceneGetEntity[Character](w, player.Character))
	char.CharacterSubstep(w, player.Character, cmd, info)
}

// TODO: make this into a proper pass
func (player Player) PlayerUpdate(w *Scene, id ecs.ID, info *UpdateParams) {
	char := mustOk(SceneGetEntity[Character](w, player.Character))
	char.CharacterUpdate(w, player.Character, info)
}
