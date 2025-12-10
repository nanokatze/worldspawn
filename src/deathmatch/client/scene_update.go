package main

import (
	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/renderer"
)

type sceneUpdate struct {
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time

	// TODO: remove renderer.SceneUpdate entirely
	*renderer.SceneUpdate
}
