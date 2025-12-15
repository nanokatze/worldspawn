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

	log.SetFlags(0)

	config.Store(&Config{})

	// We don't use SDL event watcher to handle resizes as our redraw is too
	// slow to provide responsive size changes.
	//
	// For handling input, there appears to be marginal to no benefit over using
	// WaitEvents.

	initAudio()

	initGamepad()

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
	// gamepad, _ = sdl.OpenGamepad(sdl.GetGamepads()[0])

	slog.Info("gamepad", "gamepad", gamepad)

	raddr := flag.Arg(0)

	// TODO: should newRemoteSession do the logging instead? Yes.

	game.Data = os.DirFS(*dataDir)

	session, err := newClient(gameRenderer, raddr)
	if err != nil {
		log.Fatal(err)
	}

	currentSession.Store(session)

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
