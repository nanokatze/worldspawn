package main

import (
	"time"

	"worldspawn/deathmatch/internal/game"
)

type multiRenderer []Renderer

func (multi multiRenderer) Reset(n int) {
	for _, re := range multi {
		re.Reset(n)
	}
}

func (multi multiRenderer) Update(world *game.World, playerID game.EntityID, t0, t1 game.Time, frameDuration time.Duration) {
	for _, re := range multi {
		re.Update(world, playerID, t0, t1, frameDuration)
	}
}

func (multi multiRenderer) UpdateSubtick(world *game.World, playerID game.EntityID) {
	for _, re := range multi {
		re.UpdateSubtick(world, playerID)
	}
}
