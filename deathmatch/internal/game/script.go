package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

type scriptFuncs struct {
	State reflect.Type

	// TODO: don't pass scene and id as is but instead some convenience things
	// for access control?

	Think func(scene *Scene, id ecs.ID, info *UpdateParams)

	Impact func(scene *Scene, id ecs.ID, impact Impact, info *UpdateParams)

	// Input func(scene *Scene, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams)

	// WeaponCreateProp func(scene *Scene)
	// WeaponUpdate func(scene *Scene) Recoil

	// OutOfBounds func(scene *Scene, id ecs.ID, info *UpdateParams)
}

var scripts = map[string]scriptFuncs{}

// TODO: we also need a way to SetScript and immediately initialize the state in
// a convenient manner.
func (scene *Scene) SetScript(id ecs.ID, scriptName string) {
	scene.Script.Set(id, scriptName)
	// TODO: how do we deal with the state
}

// TODO: return a pointer instead of struct as is?
func (scene *Scene) GetScriptFuncs(id ecs.ID) scriptFuncs {
	scriptName, _ := scene.Script.Get(id)
	script := scripts[scriptName]
	return script
}
