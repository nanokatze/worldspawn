package game

import (
	"reflect"
	"slices"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type PlayerSpawn struct{}

func init() {
	Scripts[reflect.TypeFor[PlayerSpawn]()] = script{}
}

// TODO: we need to pass more data here to choose the spawn point
// TODO: we should also perform collision queries to ensure free space
func (world *World) findPlayerSpawn() gmath.TRS3f64 {
	candidates := slices.Collect(func(yield func(ecs.ID) bool) {
		for id, entity := range ecs.All(&world.ScriptState) {
			if _, ok := entity.(PlayerSpawn); ok {
				yield(id)
			}
		}
	})

	rnd := Rand(world.Now) // TODO: pass info explicitly I guess

	for {
		// TODO: perform collision queries to make sure the spawn point is free.
		// The collision geometry and other info should be passed by the user.

		return world.GetGlobalTransform(candidates[rnd.IntN(len(candidates))]).TRS()
	}
}
