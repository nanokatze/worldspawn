package game

import (
	"reflect"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

// TODO: rename this
// TODO: we can factor it out now (along with UpdateParams)
// TODO: to allow for parallel enqueue we'll need mutexes too
// TODO: invalidate IO to capture uninentional capture, etc.
type IO struct {
	// TODO: IO doesn't need World, we should replace this with a collection of
	// buffers to enqueue the update funcs into
	// world   *World
	updates       ecs.Column[[]updatef]
	globalUpdates *[]func(world *World, info *UpdateParams)

	entity ecs.ID // TODO: rename to sender?
}

type updatef struct {
	from ecs.ID
	f    func(world *World, id ecs.ID, info *UpdateParams)
}

// TODO: shorter names
func (io *IO) EnqueueEntityUpdate(to ecs.ID, f func(world *World, entity ecs.ID, info *UpdateParams)) {
	updates, _ := io.updates.Get(to)
	io.updates.Set(to, append(updates, updatef{io.entity, f}))
}

func (io *IO) EnqueueGlobalUpdate(f func(world *World, info *UpdateParams)) {
	*io.globalUpdates = append(*io.globalUpdates, f)
}

// TODO: we need two variants of this: read-only one and read-write one.
/*
type Entity2 struct {
	world *World
	id    ecs.ID
}

func (e Entity2) SetTransform(transform gmath.TRS3f64) {
	e.world.SetTransform(e.id, transform)
}
*/

// TODO: replace world and id with some kind of object to enable tighter access
// control?
// TODO: pass the object through which updates are enqueued explicitly. That way
// we could straightforwardly doublebuffer things.
// TODO: rename to just script
type script struct {
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

	// TODO: rename this, this is not a hint but provides some info which is the
	// responsibility of the thing using the weapon to implement
	Weapon_Hint func(world *World, weapon ecs.ID, info *UpdateParams) WeaponHint

	Weapon_CreateProp func(world *World, weapon ecs.ID, info *UpdateParams) ecs.ID

	// TODO: I think we need to split Weapon_Think into two, one subtick/Input
	// thing and other the equivalent of Think basically.
	Weapon_Think func(
		world *World,
		weapon ecs.ID,
		props []ecs.ID,
		attacker ecs.ID,
		T_attack gmath.Affine3f64,
		v_attack Velocity,
		buttons WeaponButtons,
		info *UpdateParams) Recoil

	// TODO: we might want to specify ammo type or at least mask?
	// TODO: allow pulling multiple rounds? ideally we'd specify min and max.
	Magazine_Pull func(world *World, entity ecs.ID, ammoType AmmoType, info *UpdateParams) bool
}

var scripts = map[reflect.Type]script{}

// TODO: return a pointer instead of struct as is?
// TODO: provide a more convenient way to call functions
func (world *World) GetScriptFuncs(entity ecs.ID) script {
	typ, _ := world.Entity.Get(entity)
	return scripts[reflect.TypeOf(typ)]
}
