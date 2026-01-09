package actionset

import "worldspawn/sdl"

type ActionSet struct {
	Keys                 map[sdl.Keycode]int32
	Buttons              map[sdl.GamepadButton]int32
	RightTrigger         int32
	RightTriggerFullPull int32
	RightTriggerSoftPull int32
	LeftTrigger          int32
	LeftTriggerFullPull  int32
	LeftTriggerSoftPull  int32
}
