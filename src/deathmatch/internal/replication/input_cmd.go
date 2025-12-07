package replication

import (
	"maps"
	"reflect"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var InputCmdMarshalers = func() nice.Option {
	m1 := maps.Collect(func(yield func(reflect.Type, uint8) bool) {
		for idx, typ := range game.InputCmds {
			yield(typ, uint8(idx))
		}
	})
	m2 := maps.Collect(func(yield func(uint8, reflect.Type) bool) {
		for idx, typ := range game.InputCmds {
			yield(uint8(idx), typ)
		}
	})

	return nice.JoinOptions(
		nice.WithMarshaler(game.InterfaceNiceMarshaler[game.InputCmd](m1)),
		nice.WithUnmarshaler(game.InterfaceNiceUnmarshaler[game.InputCmd](m2)))
}()
