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

	ActionSlot0
	ActionSlot1
	ActionSlot2
	ActionSlot3

	ActionSetMovementVelocityX
	ActionSetMovementVelocityY

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
	case ActionJump, ActionCrouch, ActionAttack, ActionReload:
		if value != 0 {
			return game.InputCmdPressButton(action)
		} else {
			return game.InputCmdReleaseButton(action)
		}

	case ActionSlot0, ActionSlot1, ActionSlot2, ActionSlot3:
		// TODO: we should do nothing if value == 0
		return game.Slot(action - ActionSlot0)

	case ActionSetMovementVelocityX:
		return game.InputCmdMoveX(value)

	case ActionSetMovementVelocityY:
		return game.InputCmdMoveY(value)

	case ActionDLookX:
		return game.InputCmdDLookXY(value)

	case ActionDLookY:
		return game.InputCmdDLookYZ(value)

	default:
		panic("unknown action")
	}
}
