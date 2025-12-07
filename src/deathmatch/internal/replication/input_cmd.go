package replication

import (
	"maps"
	"reflect"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var InputCmdMarshalers = nice.JoinOptions(
	nice.WithMarshaler(game.InterfaceNiceMarshaler[game.InputCmd](
		maps.Collect(func(yield func(reflect.Type, uint8) bool) {
			for idx, typ := range game.InputCmds {
				yield(typ, uint8(idx))
			}
		}))),
	nice.WithUnmarshaler(game.InterfaceNiceUnmarshaler[game.InputCmd](
		maps.Collect(func(yield func(uint8, reflect.Type) bool) {
			for idx, typ := range game.InputCmds {
				yield(uint8(idx), typ)
			}
		}))))
