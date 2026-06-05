package game

import "worldspawn/internal/ecs"

func init() {
	// TODO: rename to delete_on_think?
	scripts["delete_after"] = scriptFuncs{
		Think: func(scene *Scene, id ecs.ID, info *UpdateParams) {
			scene.SendMessage(id,
				func(scene *Scene, id ecs.ID, info *UpdateParams) {
					scene.Delete.Set(id, struct{}{})
				})
		},
	}
}
