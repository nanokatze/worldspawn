package game

import "worldspawn/internal/ecs"

func init() {
	// TODO: rename to delete_on_think?
	scripts["delete_after"] = script{
		Think: func(world *World, entity ecs.ID, _ *UpdateParams) {
			io := IO{world, entity}

			io.EnqueueEntityUpdate(entity,
				func(world *World, entity ecs.ID, _ *UpdateParams) {
					world.Delete.Set(entity, struct{}{})
				})
		},
	}
}
