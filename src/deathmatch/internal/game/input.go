package game

// TODO: give these a type
// TODO: rename to axis?
// TODO: these depend on action sets as well
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

	ActionSetMovementVelocityX
	ActionSetMovementVelocityY

	ActionDLookX
	ActionDLookY
)

// TODO: we need a tracker object so that we can normalize value per action and
// filter things out. Or alternatively, we can just have a function that
// computes TimestampedInputCmds out of delta between two actions.
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
			return InputCmdPressButton(action)
		} else {
			return InputCmdReleaseButton(action)
		}

	case ActionSlot0, ActionSlot1, ActionSlot2, ActionSlot3:
		// TODO: we should do nothing if value == 0
		return Slot(action - ActionSlot0)

	case ActionSetMovementVelocityX:
		return InputCmdSetMovementVelocityX(value)

	case ActionSetMovementVelocityY:
		return InputCmdSetMovementVelocityY(value)

	case ActionDLookX:
		return InputCmdDLookX(value)

	case ActionDLookY:
		return InputCmdDLookY(value)

	default:
		panic("unknown action")
	}
}
