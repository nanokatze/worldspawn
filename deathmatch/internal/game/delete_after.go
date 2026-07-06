package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

// TODO: rename to DeleteOnThink?
type DeleteAfter struct{}

func init() {
	Scripts[reflect.TypeFor[DeleteAfter]()] = script{
		Think: func(_ *UpdateParams, world *World, entity ecs.ID, io IO) {
			io.EnqueueEntityUpdate(entity,
				func(_ *UpdateParams, entity Entity2, io IO) {
					entity.MarkForDeletion()
				})
		},
	}
}
