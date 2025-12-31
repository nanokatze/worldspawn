package main

import (
	"maps"
	"sync"
	"sync/atomic"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/sdl"
)

type QualityConfig struct {
	// TODO: if we ever add SPPM or other bidirectional technique, we'll
	// need two MaxBounce and RussianRouletteThreshold parameters: one for
	// the paths leaving the eye (camera) and one for paths leaving the
	// lights.

	// TODO: plop these under advanced?

	MaxBounces int32

	RussianRouletteThreshold int32
}

type Config struct {
	Presentation struct {
		Resolution [2]int64 // TODO: rename; make a Width Height struct?
	}

	Quality QualityConfig

	// https://github.com/libsdl-org/SDL/issues/4464 🥺
	//
	// TODO: rename?
	Controls struct {
		KeyActions           map[sdl.Keycode]int
		MouseButtonActions   map[sdl.MouseButton]int
		GamepadButtonActions map[sdl.GamepadButton]int
	}

	// By convention, all of these should default to their "zero" values.
	Developer struct {
		DisableCosmeticOffset bool
		DontInterpolate       bool
	}
}

func (c *Config) Clone() *Config {
	c2 := *c
	c2.Controls.KeyActions = maps.Clone(c2.Controls.KeyActions)
	c2.Controls.GamepadButtonActions = maps.Clone(c2.Controls.GamepadButtonActions)
	return &c2
}

// TODO: we need default config always accessible so that we can refer to it
// when resetting specific things (like keymaps etc.)
var defaultConfig = func() *Config {
	var conf Config

	conf.Presentation.Resolution = [2]int64{1280, 800}

	conf.Quality = QualityConfig{
		MaxBounces:               2,
		RussianRouletteThreshold: 1,
	}

	conf.Controls.KeyActions = map[sdl.Keycode]int{
		sdl.K_W:     game.ActionSetMovementVelocityY,
		sdl.K_D:     game.ActionSetMovementVelocityX,
		sdl.K_SPACE: game.ActionJump,
		sdl.K_LCTRL: game.ActionCrouch,
	}
	conf.Controls.MouseButtonActions = map[sdl.MouseButton]int{
		sdl.BUTTON_LEFT: game.ActionAttack,
	}
	conf.Controls.GamepadButtonActions = map[sdl.GamepadButton]int{
		sdl.GAMEPAD_BUTTON_DPAD_UP:    game.ActionSlot1,
		sdl.GAMEPAD_BUTTON_DPAD_DOWN:  game.ActionSlot3,
		sdl.GAMEPAD_BUTTON_DPAD_LEFT:  game.ActionSlot0,
		sdl.GAMEPAD_BUTTON_DPAD_RIGHT: game.ActionSlot2,
	}

	return &conf
}()

var config rcu[Config]

// TODO: move out of this package
type rcu[T any] struct {
	P    atomic.Pointer[T]
	WrMu sync.Mutex // only locked when modifying the value
}

func (rcu *rcu[T]) Load() *T { return rcu.P.Load() }
