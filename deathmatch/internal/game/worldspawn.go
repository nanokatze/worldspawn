package game

import (
	"reflect"
	"unique"

	"worldspawn/internal/gmath"
)

// TODO: rename to Worldspawn
type WorldGlobals struct {
	Now Time

	// TODO: replace it with sky material
	Sky unique.Handle[string]

	// TODO: create a separate "physics world" entity/component and move this
	// stuff there
	Gravity gmath.Vec3f32
}

func init() {
	Scripts[reflect.TypeFor[WorldGlobals]()] = script{}
}

func (world *World) Globals() WorldGlobals {
	globals, _ := world.GetEntity[WorldGlobals](1)
	return globals
}
