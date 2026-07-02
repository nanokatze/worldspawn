package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

type TriggerKill struct{}

func (TriggerKill) entity() {}

func init() {
	scripts[reflect.TypeFor[TriggerKill]()] = script{
		ContactAdded: func(info *UpdateParams, world *World, entity1, entity2 ecs.ID) {
			world.Delete.Set(entity2, struct{}{})
			info.Logger.Info("entity entered the trigger", "us", entity1, "them", entity2)
		},
		ContactRemoved: func(info *UpdateParams, world *World, entity1, entity2 ecs.ID) {
			info.Logger.Info("entity left the trigger", "us", entity1, "them", entity2)
		},
	}
}
