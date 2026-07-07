package game

import (
	"reflect"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

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

	// OutOfBounds func(info *UpdateParams, world *World, entity ecs.ID)

	// TODO: in Thinkers, we might want to swap the read-only world and the entity accessor (which we'll have in place of IDs)

	// TODO: prefix this somehow, e.g. with Character. Also rename to HandleInput?
	// TODO: allow this to return error, in which case the server would drop the player?
	Input func(info *UpdateParams, world *World, entity ecs.ID, cmd TimestampedInputCmd)

	// Think may not perform any mutations, but may read states of entities,
	// perform physics queries and enqueue updates.
	Think func(info *UpdateParams, world *World, entity ecs.ID, io IO)

	// Physics

	// ShouldCollide may not perform any mutations. It also may not perform
	// physics queries. Reading states of entities is fine, however.
	//
	// TODO: naming
	// TODO: pass JPH::CollideShapeResult
	ShouldCollide func(info *UpdateParams, world *World, entity1, entity2 ecs.ID) int // TODO: return a enum that corresponds to JPH::ValidateResult

	// Note that ContactAdded and ContactRemoved are not called
	// deterministically, it's thus necessary to pay extra care so that the
	// simulation is reproducible.
	//
	// TODO: naming. NewContactPair sounds like a good replacement for
	// ContactAdded but I'm not sure what to rename ContactRemoved to.
	// TODO: inout parameter which lets the script edit the contact
	// TODO: should this be thinker or mutator? I'm inclined towards the thinker...
	ContactAdded   func(info *UpdateParams, world *World, entity1, entity2 ecs.ID)
	ContactRemoved func(info *UpdateParams, world *World, entity1, entity2 ecs.ID)

	// Impact may not perform any queries, but may mutate the entity.
	Impact func(info *UpdateParams, entity Entity2, impact Impact, io IO)

	// TODO: rename this, this is not a hint but provides some info which is the
	// responsibility of the thing using the weapon to implement
	// TODO: make this a read-only thingy?
	Weapon_Hint func(info *UpdateParams, world *World, weapon ecs.ID) WeaponHint

	// TODO: rethink how weapon props should work, again
	Weapon_CreateProp func(info *UpdateParams, world *World, weapon ecs.ID) ecs.ID

	// TODO: I think we need to split Weapon_Think into two, one subtick/Input
	// thing and other the equivalent of Think basically.
	Weapon_Think func(
		info *UpdateParams,
		world *World,
		weapon ecs.ID,
		props []ecs.ID,
		attacker ecs.ID,
		T_attack gmath.Affine3f64,
		v_attack Velocity,
		buttons WeaponButtons,
		io IO) Recoil

	// TODO: we might want to specify ammo type or at least mask?
	// TODO: allow pulling multiple rounds? ideally we'd specify min and max.
	// TODO: this should not have IO but a pure mutator
	Magazine_Pull func(info *UpdateParams, entity Entity2, ammoType AmmoType, io IO) bool
}

// Public only so that replication can load/store things. We'll eventually make this private.
var Scripts = map[reflect.Type]script{}
