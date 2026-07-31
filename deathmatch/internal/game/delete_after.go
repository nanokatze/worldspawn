package game

import (
	"reflect"
)

// TODO: rename to DeleteOnThink?
type DeleteAfter struct{}

func init() {
	Scripts[reflect.TypeFor[DeleteAfter]()] = script{
		Think: func(stx ScriptContext, world *World, entity Entity2) {
			stx.Update(entity,
				func(stx ScriptContext, entity Entity2) {
					entity.MarkForDeletion()
				})
		},
	}
}
