package main

// #cgo LDFLAGS: -lphysics -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"runtime"

	"github.com/go-json-experiment/json"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/sdl"
	"worldspawn/sdlrouter"
)

func init() {
	osDirFSFlag(flag.CommandLine, &game.Data, "data", "data/cooked", "a")
}

var sdlHints = [][2]string{
	{"SDL_JOYSTICK_HIDAPI_STEAMDECK", "1"},
}

var sdlSubsystems = []sdl.InitFlags{
	sdl.INIT_AUDIO,
	sdl.INIT_VIDEO,
	sdl.INIT_GAMEPAD,
}

// TODO: this needs to live behind atomic.Pointer if we intend on being able to
// switch the localization at runtime.
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

	slog.Info("gamepads", "gamepads", sdl.GetGamepads())

	for _, gamepadID := range sdl.GetGamepads() {
		sdl.OpenGamepad(gamepadID)
	}

	go new(mainWindow).Run()

	sdlrouter.Main()
}

func osDirFSFlag(f *flag.FlagSet, p *fs.FS, name string, dir string, usage string) {
	*p = os.DirFS(dir)
	f.Func(name, usage, func(dir string) error { *p = os.DirFS(dir); return nil })
}
