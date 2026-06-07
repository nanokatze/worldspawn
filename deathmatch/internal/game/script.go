package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

// TODO: replace world and id with some kind of object to enable tighter access
// control?
// TODO: pass the object through which updates are enqueued explicitly. That way
// we could straightforwardly doublebuffer things.
type scriptFuncs struct {
	// TODO: actually nuke this and let scripts do whatever?
	State reflect.Type

	// Thinkers must not mutate any entities. Thinkers are only allowed to read
	// entity states, perform physics queries and enqueue entity updates.
	Think func(world *World, id ecs.ID, info *UpdateParams)

	Impact func(world *World, id ecs.ID, impact Impact, info *UpdateParams)

	// Input func(world *World, id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams)

	WeaponCreateProp func(world *World, info *UpdateParams) ecs.ID
	// WeaponUpdate func(world *World) Recoil

	// OutOfBounds func(world *World, id ecs.ID, info *UpdateParams)
}

var scripts = map[string]scriptFuncs{}

// TODO: we also need a way to SetScript and immediately initialize the state in
// a convenient manner.
func (world *World) SetScript(id ecs.ID, scriptID string) {
	world.Script.Set(id, scriptID)
	// TODO: how do we deal with the state
}

// TODO: return a pointer instead of struct as is?
func (world *World) GetScriptFuncs(id ecs.ID) scriptFuncs {
	scriptID, _ := world.Script.Get(id)
	return scripts[scriptID]
}
