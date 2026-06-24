package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

type DeleteAfter struct{}

func (DeleteAfter) entity() {}

func init() {
	// TODO: rename to delete_on_think?
	scripts[reflect.TypeFor[DeleteAfter]()] = script{
		Think: func(world *World, entity ecs.ID, _ *UpdateParams) {
			io := IO{world.Updates, &world.globalUpdates, entity}

			io.EnqueueEntityUpdate(entity,
				func(world *World, entity ecs.ID, _ *UpdateParams) {
					world.Delete.Set(entity, struct{}{})
				})
		},
	}
}
