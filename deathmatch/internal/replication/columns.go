package replication

import (
	"reflect"

	"worldspawn/deathmatch/internal/game"
)

// TODO: return column info or smth struct with index etc?
var Columns = reflect.TypeFor[game.Columns]()
