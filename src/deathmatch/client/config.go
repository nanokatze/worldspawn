package main

import (
	"sync"
	"sync/atomic"

	"worldspawn/sdl"
)

type Config struct {
	Resolution [2]int // TODO: rename; make a Width Height struct?

	DisableCosmeticOffset bool
	DontInterpolate       bool

	// https://github.com/libsdl-org/SDL/issues/4464 🥺

	KeyActions           map[sdl.Keycode]int
	GamepadButtonActions map[sdl.GamepadButton]int
}

var (
	configMu sync.Mutex // only locked when modifying
	config   atomic.Pointer[Config]
)

func updateConfig(f func(p *Config)) {
	configMu.Lock()
	defer configMu.Unlock()

	tmp := *config.Load()
	f(&tmp)
	config.Store(&tmp)
}
