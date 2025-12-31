package main

// #cgo LDFLAGS: -lphysics -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/go-json-experiment/json"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"worldspawn/sdl"
	"worldspawn/sdlapp"
)

var dataDir = flag.String("data", "data/cooked", "a")

var sdlHints = [][2]string{
	{"SDL_JOYSTICK_HIDAPI_STEAMDECK", "1"},
}

var sdlSubsystems = []sdl.InitFlags{
	sdl.INIT_AUDIO,
	sdl.INIT_VIDEO,
	sdl.INIT_GAMEPAD,
}

var messagePrinter = message.NewPrinter(language.English)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	flag.Parse()

	config.P.Store(defaultConfig)

	// TODO: use xdg config path
	// TODO: factor this out? this is very gross in its current state.
	if f, err := os.Open("config.json"); err == nil {
		config.WrMu.Lock()
		conf := config.Load().Clone()
		if err := json.UnmarshalRead(f, conf); err != nil {
			panic(err)
		}
		config.P.Store(conf)
		config.WrMu.Unlock()
	}

	for _, hint := range sdlHints {
		if err := sdl.SetHint(hint[0], hint[1]); err != nil {
			panic(err)
		}
	}

	for _, subsystem := range sdlSubsystems {
		if err := sdl.InitSubSystem(subsystem); err != nil {
			panic(fmt.Sprintf("failed to initialize SDL %v subsystem : %v", subsystem, err))
		}
	}

	go newMainWindow().run()

	if err := sdlapp.Main(); err != nil {
		panic(err)
	}
}
