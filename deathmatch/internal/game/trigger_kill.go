package game

import (
	"worldspawn/internal/ecs"
)

func init() {
	scripts["trigger_kill"] = script{
		ContactAdded: func(world *World, entity1, entity2 ecs.ID, info *UpdateParams) {
			world.Delete.Set(entity2, struct{}{})
			info.Logger.Info("entity entered the trigger", "us", entity1, "them", entity2)
		},
		ContactRemoved: func(world *World, entity1, entity2 ecs.ID, info *UpdateParams) {
			info.Logger.Info("entity left the trigger", "us", entity1, "them", entity2)
		},
	}
}
