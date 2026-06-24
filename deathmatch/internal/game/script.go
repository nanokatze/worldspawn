package game

import (
	"reflect"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

// TODO: go back to sort of the previous thing we had, where we'd use types to
// identify scripts. If we allow loading scripts at runtime we'd just let stuff
// register its own types I guess.

// TODO: rename this
// TODO: we can factor it out now (along with UpdateParams)
// TODO: to allow for parallel enqueue we'll need mutexes too
type IO struct {
	// TODO: IO doesn't need World, we should replace this with a collection of
	// buffers to enqueue the update funcs into
	// world   *World
	updates       ecs.Column[[]func(world *World, entity ecs.ID, info *UpdateParams)]
	globalUpdates *[]func(world *World, info *UpdateParams)

	entity ecs.ID // TODO: rename to sender?
}

// TODO: shorter names
func (io *IO) EnqueueEntityUpdate(to ecs.ID, f func(world *World, entity ecs.ID, info *UpdateParams)) {
	updates, _ := io.updates.Get(to)
	io.updates.Set(to, append(updates, f))
}

func (io *IO) EnqueueGlobalUpdate(f func(world *World, info *UpdateParams)) {
	*io.globalUpdates = append(*io.globalUpdates, f)
}

// TODO: replace world and id with some kind of object to enable tighter access
// control?
// TODO: pass the object through which updates are enqueued explicitly. That way
// we could straightforwardly doublebuffer things.
// TODO: rename to just script
type script struct {
	Type reflect.Type

	Funcs map[string]any

	// TODO: handle the state as well so that the functions don't have to poke
	// world.GetEntity

	// TODO: the following should be shadows of Funcs basically

	// OutOfBounds func(world *World, entity ecs.ID, info *UpdateParams)

	// TODO: prefix this somehow, e.g. with Character. Also rename to HandleInput?
	Input func(world *World, entity ecs.ID, cmd TimestampedInputCmd, info *UpdateParams)

	// Think may not perform any mutations, but may read states of entities,
	// perform physics queries and enqueue updates.
	Think func(world *World, entity ecs.ID, info *UpdateParams)

	// Physics

	// ShouldCollide may not perform any mutations. It also may not perform
	// physics queries. Reading states of entities is fine, however.
	//
	// TODO: naming
	// TODO: pass JPH::CollideShapeResult
	ShouldCollide func(world *World, entity1, entity2 ecs.ID, info *UpdateParams) int // TODO: return a enum that corresponds to JPH::ValidateResult

	// Note that ContactAdded and ContactRemoved are not called
	// deterministically, it's thus necessary to pay extra care so that the
	// simulation is reproducible.
	//
	// TODO: naming. NewContactPair sounds like a good replacement for
	// ContactAdded but I'm not sure what to rename ContactRemoved to.
	// TODO: inout parameter which lets the script edit the contact
	ContactAdded   func(world *World, entity1, entity2 ecs.ID, info *UpdateParams)
	ContactRemoved func(world *World, entity1, entity2 ecs.ID, info *UpdateParams)

	// Impact may not perform any queries, but may mutate the entity.
	Impact func(world *World, entity ecs.ID, impact Impact, info *UpdateParams)

	// TODO: prefix things with underscore, e.g. Weapon_CreateProp

	// TODO: elaborate that this is for FPS
	WeaponHint       func(world *World, weapon ecs.ID) WeaponHint
	WeaponCreateProp func(world *World, weapon ecs.ID, info *UpdateParams) ecs.ID
	// TODO: I think we need to split WeaponThink into two, one subtick/Input
	// thing and other the equivalent of Think basically.
	WeaponThink func(
		world *World,
		weapon ecs.ID,
		props []ecs.ID,
		attacker ecs.ID,
		T_attack gmath.Affine3f64,
		v_attack Velocity,
		buttons WeaponButtons,
		info *UpdateParams) Recoil
}

var scripts = map[string]script{}

// TODO: we also need a way to SetScript and immediately initialize the state in
// a convenient manner.
func (world *World) SetScript(entity ecs.ID, filename string) {
	world.Script.Set(entity, filename)
	if scripts[filename].Type != nil {
		world.Entity.Set(entity, reflect.Zero(scripts[filename].Type).Interface().(Entity))
	}
}

// TODO: return a pointer instead of struct as is?
func (world *World) GetScriptFuncs(entity ecs.ID) script {
	filename, _ := world.Script.Get(entity)
	return scripts[filename]
}
