package replication

import (
	"reflect"
	"slices"

	"worldspawn/deathmatch/internal/game"
)

// TODO: return column info or smth struct with index etc?

// TODO: we could make this a function of type and move this into generic
// replication code.
var ReplicatedColumns = slices.Collect(
	func(yield func(int) bool) {
		ty := reflect.TypeFor[game.Columns]()
		for i := range ty.NumField() {
			if ty.Field(i).Tag.Get("worldspawn") == "transient" {
				continue
			}
			yield(i)
		}
	})
