package main

import (
	"sync"
	"worldspawn"

	"worldspawn/experiments/actionset"
	"worldspawn/sdl"
)

var inputMu sync.Mutex

func initGamepad() {
	sdl.SetHint("SDL_JOYSTICK_HIDAPI_STEAMDECK", "1")

	if err := sdl.InitSubSystem(sdl.INIT_GAMEPAD); err != nil {
		panic(err)
	}
}

/*
const (
	ActionSetMenu = iota
	ActionSetOnFoot
)
*/

// Follow https://github.com/libsdl-org/SDL/issues/4464

var actionSets = map[string]actionset.ActionSet{
	"ON_FOOT": {
		Keys: map[sdl.Keycode]int32{
			sdl.K_SPACE: worldspawn.ActionJump,
			sdl.K_LCTRL: worldspawn.ActionCrouch,
		},
		Buttons: map[sdl.GamepadButton]int32{
			sdl.GAMEPAD_BUTTON_DPAD_UP:    worldspawn.ActionSlot1,
			sdl.GAMEPAD_BUTTON_DPAD_DOWN:  worldspawn.ActionSlot3,
			sdl.GAMEPAD_BUTTON_DPAD_LEFT:  worldspawn.ActionSlot0,
			sdl.GAMEPAD_BUTTON_DPAD_RIGHT: worldspawn.ActionSlot2,
		},
		RightTriggerFullPull: worldspawn.ActionAttack,
	},
}
