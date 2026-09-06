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

	Mode string // TODO: this should be an interface/enum basically
}

func init() {
	Scripts[reflect.TypeFor[Worldspawn]()] = script{}
}
