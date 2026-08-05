package game

import (
	"reflect"
	"slices"
)

var ReplicatedColumns = slices.Collect(
	func(yield func(int) bool) {
		ty := reflect.TypeFor[Columns]()
		for i := range ty.NumField() {
			if ty.Field(i).Tag.Get("worldspawn") == "dontreplicate" {
				continue
			}
			yield(i)
		}
	})
