package replication

import (
	"fmt"
	"reflect"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var NiceOptions = nice.WithArshalers(nice.JoinArshalers(
	nice.MakeUniqueHandleArshaler[string](),
	nice.MakeInterfaceArshaler2[game.ScriptState](
		func(typ reflect.Type) string {
			if _, ok := game.Scripts[typ]; !ok {
				panic(fmt.Sprintf("bad %#v", typ))
			}
			return typ.Name()
		},
		func(typId string) reflect.Type {
			for typ := range game.Scripts {
				if typ.Name() == typId {
					return typ
				}
			}
			panic(fmt.Sprintf("bad %v", typId))
		}),
	// TODO: kill this
	nice.MakeInterfaceArshaler[game.InputCmd](game.InputCmdTypes...)))
