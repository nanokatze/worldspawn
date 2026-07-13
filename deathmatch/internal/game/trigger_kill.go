package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

type TriggerKill struct{}

func init() {
	Scripts[reflect.TypeFor[TriggerKill]()] = script{
		ContactAdded: func(info *UpdateParams, world *World, entity1, entity2 ecs.ID) {
			world.GetEntity2(entity2).MarkForDeletion()
			world.logger.Info("entity entered the trigger", "us", entity1, "them", entity2)
		},
		ContactRemoved: func(info *UpdateParams, world *World, entity1, entity2 ecs.ID) {
			world.logger.Info("entity left the trigger", "us", entity1, "them", entity2)
		},
	}
}
