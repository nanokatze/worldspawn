package game

import (
	"reflect"

	"worldspawn/internal/ecs"
)

// TODO: replace world and id with some kind of object to enable tighter access
// control?
// TODO: pass the object through which updates are enqueued explicitly. That way
// we could straightforwardly doublebuffer things.
// TODO: rename to just script
type scriptFuncs struct {
	// Types to register for de/serialization
	Types []reflect.Type

	Funcs map[string]any

	// TODO: the following should be shadows of Funcs basically

	// Think may not perform any mutations, but may read states of entities,
	// perform physics queries and enqueue updates.
	Think func(world *World, entity ecs.ID, info *UpdateParams)

	// Impact may not perform any queries, but may mutate the entity.
	Impact func(world *World, entity ecs.ID, impact Impact, info *UpdateParams)

	Input func(world *World, entity ecs.ID, cmd TimestampedInputCmd, info *UpdateParams)

	WeaponCreateProp func(world *World, info *UpdateParams) ecs.ID
	// WeaponUpdate func(world *World) Recoil

	// Physics

	// ShouldCollide may not perform any mutations. It also may not perform
	// physics queries. Reading states of entities is fine, however.
	//
	// TODO: naming
	// TODO: pass JPH::CollideShapeResult
	ShouldCollide func(world *World, entity1, entity2 ecs.ID) int // TODO: return a enum that corresponds to JPH::ValidateResult

	// Note that ContactAdded and ContactRemoved are not called
	// deterministically, it's thus necessary to put extra care so that the
	// simulation is reproducible.
	//
	// TODO: naming
	// TODO: inout parameter which lets the script edit the contact
	ContactAdded func(world *World, entity1, entity2 ecs.ID)
	// ContactRemoved func(world *World)

	// OutOfBounds func(world *World, entity ecs.ID, info *UpdateParams)
}

var scripts = map[string]scriptFuncs{}

// TODO: we also need a way to SetScript and immediately initialize the state in
// a convenient manner.
func (world *World) SetScript(entity ecs.ID, scriptID string) {
	world.Script.Set(entity, scriptID)
	// TODO: how do we deal with the state
}

// TODO: return a pointer instead of struct as is?
// TODO: rename to GetScript
func (world *World) GetScriptFuncs(entity ecs.ID) scriptFuncs {
	scriptID, _ := world.Script.Get(entity)
	return scripts[scriptID]
}
