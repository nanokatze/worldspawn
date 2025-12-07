package replication

import (
	"maps"
	"reflect"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var NiceOptions = nice.JoinOptions(
	nice.WithMarshaler(game.InterfaceNiceMarshaler[game.Entity](
		maps.Collect(func(yield func(reflect.Type, uint64) bool) {
			for idx, typ := range game.EntityTypes {
				yield(typ, uint64(idx)) // TODO: just hash instead of idx?
			}
		}))),
	nice.WithUnmarshaler(game.InterfaceNiceUnmarshaler[game.Entity](
		maps.Collect(func(yield func(uint64, reflect.Type) bool) {
			for idx, typ := range game.EntityTypes {
				yield(uint64(idx), typ)
			}
		}))))
