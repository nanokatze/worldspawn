package game

import (
	"reflect"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type ScriptContext struct {
	*UpdateParams
	IO
}

// TODO: all mutators should be continuation-based for composability. I.e.
// instead of returning a result, they should take a continuation to call.

// TODO: switch over more funcs to use Entity2?
type script struct {
	Funcs map[string]any

	// TODO: handle the state as well so that the functions don't have to poke
	// world.GetEntity

	// TODO: the following should be shadows of Funcs basically

	// OutOfBounds func(stx ScriptContext, world *World, entity ecs.ID)

	// TODO: rename to HandleInput?
	// TODO: allow this to return error, in which case the server would drop the player?
	// TODO: this should be a Thinker and get IO. Client will just call
	// processUpdates immediately. We could expose two HandleInputs, one would
	// be for server and one for client.
	Input func(info *UpdateParams, world *World, entity ecs.ID, cmd InputCmd)

	// Think may not perform any mutations, but may read states of entities,
	// perform physics queries and enqueue updates.
	Think func(stx ScriptContext, world *World, entity Entity)

	// Physics

	// ShouldCollide may not perform any mutations. It also may not perform
	// physics queries. Reading states of entities is fine, however.
	//
	// TODO: naming
	// TODO: pass JPH::CollideShapeResult
	ShouldCollide func(stx ScriptContext, world *World, entity1, entity2 ecs.ID) int // TODO: return a enum that corresponds to JPH::ValidateResult

	// Note that ContactAdded and ContactRemoved are not called
	// deterministically, it's thus necessary to pay extra care so that the
	// simulation is reproducible.
	//
	// TODO: naming. NewContactPair sounds like a good replacement for
	// ContactAdded but I'm not sure what to rename ContactRemoved to.
	// TODO: inout parameter which lets the script edit the contact
	// TODO: should this be thinker or mutator? I'm inclined towards the thinker...
	ContactAdded   func(stx ScriptContext, world *World, entity1, entity2 ecs.ID)
	ContactRemoved func(stx ScriptContext, world *World, entity1, entity2 ecs.ID)

	// Impact may not perform any queries, but may mutate the entity.
	Impact func(stx ScriptContext, entity Entity, impact Impact)

	// TODO: rename this, this is not a hint but provides some info which is the
	// responsibility of the thing using the weapon to implement
	Weapon_Hint func(info *UpdateParams, world *World, weapon ecs.ID) WeaponHint

	Weapon_CreateProp func(stx ScriptContext, weapon Entity, f func(stx ScriptContext, prop Entity))

	// TODO: I think we need to split Weapon_Think into two, one subtick/Input
	// thing and other the equivalent of Think basically.
	// TODO: pass a continuation for recoil
	Weapon_Think func(
		stx ScriptContext,
		world *World,
		weapon Entity,
		weaponProps []Entity,
		attacker Entity,
		T_attack gmath.Affine3f64,
		v_attack Velocity,
		buttons WeaponButtons,
		// recoil func(stx ScriptContext, weapon Entity2, recoil [2]float32),
	)

	// TODO: for composability reasons, this should probably take a continuation
	Magazine_Pull func(stx ScriptContext, entity Entity, ammoType AmmoType, min, max int) int
}

// Public only so that replication can load/store things. We'll eventually make this private.
var Scripts = map[reflect.Type]script{}

// TODO: remove this, the users should look up the script associated with a
// particular script state by themselves
func (e Entity) Script() script {
	return Scripts[reflect.TypeOf(e.world.ScriptState.Load(e.id.Index()))]
}
