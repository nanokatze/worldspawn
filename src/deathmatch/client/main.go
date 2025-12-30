package main

// #cgo LDFLAGS: -lphysics -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/go-json-experiment/json"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"worldspawn/sdl"
	"worldspawn/sdlapp"
)

var dataDir = flag.String("data", "data/cooked", "a")

// TODO: should this be in worldspawn
var messagePrinter = message.NewPrinter(language.English)

var currentSession atomic.Pointer[Client]

var gamepad *sdl.Gamepad

// TODO: put sdl inits behind sync.Onces?

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	flag.Parse()

	log.SetFlags(0) // TODO: kill this line

	config.P.Store(defaultConfig)

	// log.Println(os.Hostname())

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

	initAudio()

	// TODO: check and report error
	sdl.SetHint("SDL_JOYSTICK_HIDAPI_STEAMDECK", "1")

	if err := sdl.InitSubSystem(sdl.INIT_GAMEPAD); err != nil {
		panic(err)
	}

	if err := sdl.InitSubSystem(sdl.INIT_VIDEO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL video subsystem: %v", err))
	}

	// TODO: factor stuff into mainWindow constructor

	go runMainWindow()

	if err := sdlapp.Main(); err != nil {
		panic(err)
	}
}
