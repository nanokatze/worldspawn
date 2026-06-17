package main

import (
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/ecs"
)

type multiRenderer []Renderer

func (multi multiRenderer) Reset(n int) {
	for _, re := range multi {
		re.Reset(n)
	}
}

func (multi multiRenderer) Tick(world *game.World, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	for _, re := range multi {
		re.Tick(world, playerID, t0, t1, frameDuration)
	}
}

func (multi multiRenderer) Subtick(world *game.World, playerID ecs.ID) {
	for _, re := range multi {
		re.Subtick(world, playerID)
	}
}
