package game

import (
	"reflect"
	"unique"

	"worldspawn/internal/gmath"
)

type Worldspawn struct {
	Now Time

	Sky unique.Handle[string]

	Gravity gmath.Vec3f32
}

func init() {
	Scripts[reflect.TypeFor[Worldspawn]()] = script{}
}
