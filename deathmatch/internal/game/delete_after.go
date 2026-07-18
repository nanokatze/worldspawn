package game

import (
	"reflect"
)

// TODO: rename to DeleteOnThink?
type DeleteAfter struct{}

func init() {
	Scripts[reflect.TypeFor[DeleteAfter]()] = script{
		Think: func(_ *UpdateParams, world *World, entity Entity2, io IO) {
			io.Update(entity,
				func(_ *UpdateParams, entity Entity2, io IO) {
					entity.MarkForDeletion()
				})
		},
	}
}
