package main

import "worldspawn/deathmatch/internal/game"

// TODO: actions can be roughly partitioned into two categories: ones will
// translate to InputCmd buttons and analog inputs, and the other are more
// abstract that are handled within the client. We might wanna think about how
// to handle these.

const (
	_ = iota

	// TODO: prefix actions by the set they belong to?

	// TODO: menu actions

	ActionJump
	ActionCrouch
	ActionAttack
	ActionReload
	ActionDash

	ActionSlot0
	ActionSlot1
	ActionSlot2
	ActionSlot3

	// TODO: kill these in favor of ActionMoveX, Y
	ActionMoveBack
	ActionMoveForward
	ActionMoveLeft
	ActionMoveRight

	ActionMoveX
	ActionMoveY

	ActionDLookX
	ActionDLookY

	// TODO: more abstract actions and
)

// TODO: we need a tracker object so that we can normalize value per action and
// filter things out. Or alternatively, we can just have a function that
// computes TimestampedInputCmds out of delta between two actions.
func AppendAction(dst []game.TimestampedInputCmd, time game.Time, action int, value float32) []game.TimestampedInputCmd {
	cmd := actionToInputCmd(action, value)
	if cmd != nil {
		dst = append(dst, game.TimestampedInputCmd{Time: time, Cmd: cmd})
	}
	return dst
}

// TODO: with some extra effort we can make InputCmd values private
func actionToInputCmd(action int, value float32) game.InputCmd {
	switch action {
	case ActionJump:
		return game.InputCmdJump(value != 0)
	case ActionCrouch:
		return game.InputCmdCrouch(value != 0)
	case ActionAttack:
		return game.InputCmdAttack(value != 0)
	case ActionReload:
		return game.InputCmdReload(value != 0)
	case ActionDash:
		return game.InputCmdDash(value != 0)

	case ActionSlot0, ActionSlot1, ActionSlot2, ActionSlot3:
		// TODO: we should do nothing if value == 0
		return game.InputCmdSwitchWeapon(action - ActionSlot0)

	case ActionMoveBack:
		return game.InputCmdMoveY(-value)
	case ActionMoveForward:
		return game.InputCmdMoveY(value)
	case ActionMoveLeft:
		return game.InputCmdMoveX(-value)
	case ActionMoveRight:
		return game.InputCmdMoveX(value)
	case ActionMoveX:
		return game.InputCmdMoveX(value)
	case ActionMoveY:
		return game.InputCmdMoveY(value)

	case ActionDLookX:
		return game.InputCmdDLookXY(value)

	case ActionDLookY:
		return game.InputCmdDLookYZ(value)

	default:
		panic("unknown action")
	}
}
