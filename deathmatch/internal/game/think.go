package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

func think(world *World, updateParams *UpdateParams) {
	// TODO: optimize the pass over thinkers by having a shadow column

	for id, state := range ecs.All(&world.ScriptState) {
		script := Scripts[reflect.TypeOf(state)]
		if script.Think == nil {
			continue
		}

		// TODO: we'll want a timer wheel of sorts to make this fast
		nextThink, _ := world.NextThink.Get(id)
		if nextThink.Compare(updateParams.Now) > 0 {
			continue
		}

		script.Think(ScriptContext{updateParams, IO{world, uint64(id.Index())}}, Entity{entities: &world.Entities, id: id}, world)
	}
}
