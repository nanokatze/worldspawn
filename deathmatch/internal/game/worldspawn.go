package game

import (
	"reflect"
	"unique"

	"worldspawn/internal/gmath"
)

type Worldspawn struct {
	Now Time

	// TODO: replace it with sky material
	Sky unique.Handle[string]

	// TODO: create a separate "physics world" entity/component and move this
	// stuff there
	Gravity gmath.Vec3f32
}

func init() {
	Scripts[reflect.TypeFor[Worldspawn]()] = script{}
}

// TODO: kill this method, it's useless
func (world *World) Globals() Worldspawn {
	return world.GetEntity2(1).ScriptState().(Worldspawn)
}
