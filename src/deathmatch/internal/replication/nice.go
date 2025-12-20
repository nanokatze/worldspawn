package replication

import (
	"maps"
	"reflect"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
	"worldspawn/internal/nice2"
)

var NiceOptions = nice.JoinOptions(
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
		}))),
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

var Nice2Arshalers = nice2.MakeArshalerMap(
	nice2.WithInterfaceArshaler[game.Entity](game.EntityTypes),
	nice2.WithInterfaceArshaler[game.InputCmd](game.InputCmds))
