package main

import (
	"worldspawn/sdl"
)

func initGamepad() {
	sdl.SetHint("SDL_JOYSTICK_HIDAPI_STEAMDECK", "1")

	if err := sdl.InitSubSystem(sdl.INIT_GAMEPAD); err != nil {
		panic(err)
	}
}

// https://github.com/libsdl-org/SDL/issues/4464
