package main

// #cgo LDFLAGS: -lphysics -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/go-json-experiment/json"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/sdl"
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

	config.Store(defaultConfig)

	// TODO: use xdg config path
	// TODO: factor this out? this is very gross in its current state.
	if f, err := os.Open("config.json"); err == nil {
		configMu.Lock()
		conf := config.Load().Clone()
		if err := json.UnmarshalRead(f, conf); err != nil {
			panic(err)
		}
		config.Store(conf)
		configMu.Unlock()
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

	mainWindow := newMainWindow()

	go func() {
		for {
			mainWindow.redraw()
		}
	}()

	if err := mainWindow.sdlWindow.SetWindowRelativeMouseMode(true); err != nil {
		slog.Warn("failed to set relative mouse mode", "err", err)
	}

	slog.Info("gamepads", "gamepads", sdl.GetGamepads())

	// TODO: open all gamepads we have here
	if gamepads := sdl.GetGamepads(); len(gamepads) > 0 {
		gamepad, _ = sdl.OpenGamepad(gamepads[0])
	}

	slog.Info("gamepad", "gamepad", gamepad)

	raddr := flag.Arg(0)

	// TODO: should newRemoteSession do the logging instead? Yes.

	game.Data = os.DirFS(*dataDir)

	session, err := newClient(gameRenderer, raddr)
	if err != nil {
		log.Fatal(err)
	}

	currentSession.Store(session)

	// We don't use SDL event watcher to handle resizes as our redraw is too
	// slow to provide responsive size changes.
	//
	// For handling input, there appears to be marginal to no benefit over using
	// WaitEvents.

eventLoop:
	for {
		e, err := sdl.WaitEvent()
		if err != nil {
			panic(fmt.Sprintf("sdl.WaitEvent: %v", err))
		}

		switch e := e.(type) {
		case *sdl.QuitEvent:
			break eventLoop

		default:
			mainWindow.handleInput(e)
		}
	}
}
