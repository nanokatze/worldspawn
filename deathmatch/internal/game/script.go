package game

import (
	"reflect"

	"worldspawn/internal/gmath"
)

// TODO: replace some uses of term "Script" with "Class" actually?

// TODO: think about what we should and should not embed
type ScriptContext struct {
	*UpdateParams
	IO
}

// TODO: all mutators should be continuation-based for composability. I.e.
// instead of returning a result, they should take a continuation to call.

type script struct {
	Funcs map[string]any

	// TODO: handle the state as well so that the functions don't have to poke
	// world.GetEntity

	// TODO: the following should be shadows of Funcs basically

	// OutOfBounds func(stx ScriptContext, world *World, entity EntityID)

	// TODO: allow this to return error, in which case the server would drop the player?
	// TODO: this should be a Thinker and get IO. Client will just call
	// processUpdates immediately.
	HandleInput func(stx ScriptContext, entity Entity, world *World, cmd TimestampedInputCmd)

	// Think may not perform any mutations, but may read states of entities,
	// perform physics queries and enqueue updates.
	Think func(stx ScriptContext, entity Entity, world *World)

	// Physics

	// ShouldCollide may not perform any mutations. It also may not perform
	// physics queries. Reading states of entities is fine, however.
	//
	// TODO: naming
	// TODO: pass JPH::CollideShapeResult
	ShouldCollide func(stx ScriptContext, entity, entity2 Entity) int // TODO: return a enum that corresponds to JPH::ValidateResult

	// Note that HandleContact and HandleSeparation are not called
	// deterministically, it's thus necessary to pay extra care so that the
	// simulation is reproducible.
	//
	// TODO: inout parameter which lets the script edit the contact?
	// TODO: should this be thinker or mutator? I'm inclined towards the thinker...
	HandleContact    func(stx ScriptContext, entity, entity2 Entity)
	HandleSeparation func(stx ScriptContext, entity Entity, entity2 EntityID)

	// HandleImpact may not perform any queries, but may mutate the entity.
	HandleImpact func(stx ScriptContext, entity Entity, impact Impact)

	// TODO: rename this, this is not a hint but provides some info which is the
	// responsibility of the thing using the weapon to implement
	// TODO: this should not take any arguments mayhaps
	Weapon_Hint func(info *UpdateParams, weapon Entity) WeaponHint

	Weapon_CreateProp func(stx ScriptContext, weapon Entity, f func(stx ScriptContext, prop Entity))

	// TODO: I think we need to split Weapon_Think into two, one subtick/Input
	// thing and other the equivalent of Think basically.
	Weapon_Think func(
		stx ScriptContext,
		weapon Entity,
		weaponProps []Entity,
		world *World,
		attacker Entity,
		T_attack gmath.Affine3f64,
		V_attack Screw3,
		buttons WeaponButtons,
		recoil func(stx ScriptContext, recoil [2]float32),
	)

	// TODO: for composability reasons, this should probably take a continuation
	Magazine_Pull func(stx ScriptContext, entity Entity, ammoType AmmoType, min, max int) int
}

// Public only so that replication can load/store things. We'll eventually make this private.
var Scripts = map[reflect.Type]script{}
