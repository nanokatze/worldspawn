package game

import "worldspawn/internal/ecs"

func init() {
	// TODO: rename to delete_on_think?
	scripts["delete_after"] = script{
		Think: func(world *World, id ecs.ID, _ *UpdateParams) {
			world.EnqueueEntityUpdate(id,
				func(world *World, id ecs.ID, _ *UpdateParams) {
					world.Delete.Set(id, struct{}{})
				})
		},
	}
}
