package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

// TODO: rename to DeleteOnThink?
type DeleteAfter struct{}

func (DeleteAfter) entity() {}

func init() {
	scripts[reflect.TypeFor[DeleteAfter]()] = script{
		Think: func(_ *UpdateParams, world *World, entity ecs.ID, io IO) {
			io.EnqueueEntityUpdate(entity,
				func(_ *UpdateParams, entity ecs.ID, io IO) {
					io.world.Delete.Set(entity, struct{}{})
				})
		},
	}
}
