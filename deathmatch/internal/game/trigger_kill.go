package game

import (
	"reflect"
)

type TriggerKill struct{}

func init() {
	Scripts[reflect.TypeFor[TriggerKill]()] = script{
		ContactAdded: func(stx ScriptContext, entity1, entity2 Entity) {
			entity2.MarkForDeletion()
			stx.logger.Info("entity entered the trigger", "us", entity1, "them", entity2)
		},
		ContactRemoved: func(stx ScriptContext, entity1 Entity, entity2 EntityID) {
			stx.logger.Info("entity left the trigger", "us", entity1, "them", entity2)
		},
	}
}
