package game

import (
	"fmt"
	"io/fs"
	"reflect"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: actually indeed stick it onto Scene or pass it through UpdateInfo
var Data fs.FS

// TODO: split this file up

// TODO: kill this (again) and make it a thing on World directly. We need this
// so that we can allow querying these values even in entity updates
type WorldGlobals struct {
	// TODO: replace it with sky material
	Sky string

	// TODO: create a separate "physics world" entity/component and move this
	// stuff there
	Gravity gmath.Vec3f32
}

func (WorldGlobals) entity() {}

type TR3f64 struct {
	T gmath.Vec3f64
	R gmath.Rot3
}

// TODO: fold almost all scripty stuff into the same column? Although I guess
// that's more up to how we wanna represent things for authoring etc. I'm not
// sure it would make sense to fold animation graph script into the same column
// as game logic script. However...

// TODO: add a type for representing non-mutable lists and stuff?

// TODO: certain columns could've perfectly feasibly been unique.Handle[string].
// E.g. Skeleton, CollisionGeometry and RenderingGeometry.
// TODO: get rid of transient columns altogether (for now), they should be
// simpler structures at the root of World
// TODO: instead of ecs.Columns, these should be any objects implementing an
// interface with Get/Set/iter. We'll eventually replace ecs.Column with simpler
// structures indexed by plain integers and get rid of ecs.Table, bringing
// entity ID validation here.
type Columns struct {
	Name ecs.Column[string]

	// Entity programmability

	// TODO: rename this pls
	Entity ecs.Column[Entity]

	NextThink ecs.Column[Time]

	// Entities marked for deletion
	// TODO: doesn't really need to be a column either
	Delete ecs.Column[struct{}] `worldspawn:"transient"`

	// TODO: require that entities that we're do collision/physics for have no
	// parent. Or maybe allow that somehow, e.g. by having children be joined
	// with the collision geometry of the parent?

	// Transform

	// Do not access this column directly; use {Get,Set}Parent. Specifies the
	// parent entity.
	Parent ecs.Column[ecs.ID]
	// The bone in the parent's skeleton that transforms this entity.
	ParentBone ecs.Column[string]
	// Do not access this column directly; use {Get,Set}Transform or
	// GetGlobalTransform instead.
	//
	// The translation and rotation parts of the entity's parent-relative
	// transform.
	TransformTR ecs.Column[TR3f64]
	// Do not access this column directly; use {Get,Set}Transform and
	// GetGlobalTransform instead.
	//
	// The scale and shearing part of the entity's parent-relative transform.
	//
	// It is possible for an entity to have a TransformTR but not TransformS, in
	// which case no scaling or shearing is applied to the entity.
	// TODO: we'll later hard-require TransformS to always be set to something valid.
	TransformS ecs.Column[gmath.Mat3x3Uf32]

	// TODO: make skeletons part of geometry? Ok actually wait which geometry
	// lol. We'll need to think more about this.
	Skeleton ecs.Column[string]

	// TODO: this should be a non-networked column that would be populated by
	// the animation script
	Pose ecs.Column[animgraph.Pose]

	// Collision

	CollisionLayer    ecs.Column[CollisionLayer]
	CollisionGeometry ecs.Column[string]

	CollisionSensor ecs.Column[struct{}]

	// Motion

	// MotionType       ecs.Column[MotionType]
	// MotionProperties ecs.Column[MotionProperties]

	Velocity ecs.Column[Velocity]

	// TODO: remove GravityFactor, Physics{Mass,Inertia}Override in favor of the
	// MotionProperties column. Or better yet, CollisionGeometry script setting
	// that.

	GravityFactor          ecs.Column[float32]
	PhysicsMassOverride    ecs.Column[float32]
	PhysicsInertiaOverride ecs.Column[gmath.Mat4x4f32]

	// TODO: kill this column and handle it at prefab instantination
	CollectionInstance ecs.Column[CollectionInstance]

	// Other gameplay stuff

	// Whether the fuse of certain projectiles should be set off when they
	// collide with this entity.
	ShouldSetOffFuseOnImpact ecs.Column[struct{}]

	// Renderer

	VisibilityCondition ecs.Column[VisibilityCondition]

	CosmeticOffset ecs.Column[CosmeticOffset]

	RenderingGeometry ecs.Column[string]

	// TODO: so the way we'll probably go about sounds is by having 2 networked
	// columns and one shadow column. The shadow column will feed the actual
	// audio player, so it should contain basically sound files and t0 (a mix of
	// game time and sample count) and the networked columns be the "audio
	// program" identifier and audio program state. For now we can start with
	// just the shadow column.

	// TODO: rethink sounds
	SoundEffect      ecs.Column[SoundEmitter]
	SoundEffectState ecs.Column[LoopedSound] // TODO: kill this column

	// Misc

	// Speculative object deadline
	// TODO: rename
	Speculation ecs.Column[Time] // TODO: do not network this
}

// TODO: make all of the internals private. We'll need to add infrastructure for
// replication to World first.
type World struct {
	// TODO: could we move this to somewhere? Either way I would prefer if
	// things would consult UpdateParams.Now rather than Scene.Params
	Now Time

	NextID            ecs.ID
	NextIDSpeculative ecs.ID

	Table *ecs.Table
	Columns

	entityUpdates, entityUpdates2 [][]updatef
	globalUpdates                 []func(*UpdateParams, *World)

	physics *physics.System
	// TODO: this should be folded into physicsSystem
	physicsBodyExists ecs.Column[struct{}]
}

func NewWorld(n int) *World {
	world := new(World)

	world.Table = ecs.NewTable(n)

	// TODO: make it clear that these are reflect references

	columns := reflect.ValueOf(&world.Columns).Elem()
	for i := range columns.Type().NumField() {
		columns.Field(i).Addr().Interface().(interface{ Init(*ecs.Table) }).Init(world.Table)
	}

	world.entityUpdates = make([][]updatef, n)
	world.entityUpdates2 = make([][]updatef, n)

	// TODO: pass contact listener. Or make it so that the contact listener is
	// passed at call to Update.
	world.physics = physics.NewSystem(
		int(numCollisionLayers),
		collisionLayerToBroadPhaseLayer[:],
		collisionLayerRules)
	world.physicsBodyExists.Init(world.Table)

	// TODO: we should expose an OptimizeBroadPhase call on physicsSystem which
	// we'll (optionally) call after loading the world and perhaps every so
	// often

	return world
}

func (world *World) Cap() int { return world.Table.IDs().Cap() }

func (world *World) EntityExists(id ecs.ID) bool { return world.Table.IDs().Exists(id) }

// TODO: make this private?
// TODO: return ecs.ID. We could make CreateEntity take a lambda which will be
// called with Entity2, but it should definitely return ecs.ID.
func (world *World) CreateEntity(info *UpdateParams) Entity2 {
	// TODO: don't hardcode index ranges

	if info.Speculating {
		id := world.Table.CreateRowAuto(900, 999, &world.NextIDSpeculative)
		world.Speculation.Set(id, world.Now)
		return Entity2{world, id}
	}

	id := world.Table.CreateRowAuto(1, 899, &world.NextID)
	return Entity2{world, id}
}

// TODO: rename to ResetEntity?
func (world *World) ClearEntity(id ecs.ID) {
	if _, ok := world.physicsBodyExists.Get(id); ok {
		world.physics.RemoveBody(physics.BodyID(id.Index()))
	}
	world.Table.ClearRow(id)
}

// This is used by client networking to remove entities.
// TODO: kill
func (world *World) DeleteEntityImmediately(id ecs.ID) {
	if _, ok := world.physicsBodyExists.Get(id); ok {
		world.physics.RemoveBody(physics.BodyID(id.Index()))
	}
	world.Table.DeleteRow(id)
}

func (world *World) Globals() WorldGlobals {
	globals, _ := world.GetEntity[WorldGlobals](1)
	return globals
}

// TODO: kill all of these accessors and make them more contextual (e.g. hang onto IO or whatever)

// TODO: rename this
func (world *World) GetEntity[T Entity](id ecs.ID) (T, bool) {
	entity, _ := world.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
}

func (world *World) MutateEntity[T Entity](id ecs.ID, f func(v *T)) {
	v, _ := world.GetEntity[T](id)
	f(&v)
	world.Entity.Set(id, v)
}

func (world *World) GetParent(id ecs.ID) ecs.ID {
	parent, _ := world.Parent.Get(id)
	return parent
}

func (world *World) SetParent(id, parent ecs.ID) {
	if parent != 0 {
		world.Parent.Set(id, parent)
	} else {
		// TODO: our panic behavior should be the same as if we tried to Set
		// this id. I.e. if this id isn't valid, we should crash.
		world.Parent.Delete(id)
	}

	// children, _ := scene.Children.Load(parent)
	// children
}

func (world *World) GetTransform(id ecs.ID) gmath.TRS3f64 {
	tr, ok := world.TransformTR.Get(id)
	if !ok {
		// TODO: replace this panic with an oops-esque thing which can be just a
		// warning or whatever if need be, or return error from GetTransform and
		// have a helper on UpdateParams.
		panic(fmt.Sprintf("transform queried but does not exist on an object id=%d", id))
		return gmath.TRS3One[float64]()
	}
	s, ok := world.TransformS.Get(id)
	if !ok {
		s = gmath.Mat3x3UOne[float32]()
	}
	return gmath.TRS3f64{tr.T, tr.R, s}
}

func (world *World) SetTransform(id ecs.ID, T gmath.TRS3f64) {
	world.TransformTR.Set(id, TR3f64{T.T, T.R})
	if T.S != gmath.Mat3x3UOne[float32]() {
		world.TransformS.Set(id, T.S)
	} else {
		world.TransformS.Delete(id)
	}
}

// TODO: make this a global thing that takes IO (this needs arbitrary scene read)
//
// TODO: if we encounter errors during hierarchy traversal we should restart
// traversal with diagnostics collection and print the collected diagnostics
// after using Scene.Logger.Error
//
// TODO: cycle detection
//
// TODO: replace T.Mul(A) with just A on the first iteration to optimize the
// common case
//
// TODO: clean up
func (world *World) GetGlobalTransform(id ecs.ID) gmath.Affine3f64 {
	getEntityTransform := func(id ecs.ID) gmath.Affine3f64 {
		tr, ok := world.TransformTR.Get(id)
		if !ok {
			return gmath.Affine3One[float64]()
		}
		s, ok := world.TransformS.Get(id)
		if !ok {
			s = gmath.Mat3x3UOne[float32]()
		}
		return gmath.TRS3f64{tr.T, tr.R, s}.Compose()
	}

	// TODO: make this a method on the scene?
	getBoneTransform := func(id ecs.ID, bone string) gmath.Affine3f32 {
		skelly := world.GetSkeleton(id)
		if skelly == nil {
			return gmath.Affine3One[float32]()
		}
		boneIndex := skelly.JointByName(bone)
		if boneIndex == -1 {
			return gmath.Affine3One[float32]()
		}

		pose, _ := world.Pose.Get(id)
		boneTransform, ok := pose.Bones[boneIndex]
		if !ok {
			return skelly.BindPose[boneIndex]
		}
		return boneTransform.Mul(skelly.BindPose[boneIndex])
	}

	// TODO: don't hardcode the hierarchy depth bound
	// TODO: actually maybe have a bloom filter/small hashmap to track cycles?
	// It would be nice to avoid having a different behavior regardless of
	// whether we have cycle detection on or not.
	//
	// NOTE: the hierarchy depth is bounded by no. of entries in the table

	A := gmath.Affine3One[float64]()
	for range 5000 {
		A = getEntityTransform(id).Mul(A)

		parent := world.GetParent(id)
		if parent == 0 {
			// TODO: ensure that parent to bone isn't set
			break
		}

		if parentBone, parentedToBone := world.ParentBone.Get(id); parentedToBone {
			A = getBoneTransform(parent, parentBone).Convert[float64]().Mul(A)
		}

		id = parent
	}

	return A
}

func (world *World) GetSkeleton(id ecs.ID) *animgraph.Skeleton {
	skellyName, ok := world.Skeleton.Get(id)
	if !ok {
		return nil
	}
	return skeleton(skellyName)
}

// Entity2 must not be stored in any structures and also not passed across entity update lambdas.
//
// TODO: rename to something else
type Entity2 struct {
	world *World
	id    ecs.ID // TODO: replace with index
}

// TODO: rename to something nicer
func (e Entity2) Clear() { e.world.ClearEntity(e.id) }

// TODO: replace with an easy to use thingy for calling functions?
func (e Entity2) Script() script { return e.world.GetScriptFuncs(e.id) }

func (e Entity2) ScriptState() Entity { v, _ := e.world.Entity.Get(e.id); return v }

func (e Entity2) SetScriptState(v Entity) { e.world.Entity.Set(e.id, v) }

func (e Entity2) SetNextThink(v Time) { e.world.NextThink.Set(e.id, v) }

func (e Entity2) SetParent(v ecs.ID) { e.world.SetParent(e.id, v) }

func (e Entity2) Transform() gmath.TRS3f64 { return e.world.GetTransform(e.id) }

func (e Entity2) SetTransform(v gmath.TRS3f64) { e.world.SetTransform(e.id, v) }

func (e Entity2) SetSkeleton(v string) { e.world.Skeleton.Set(e.id, v) }

func (e Entity2) SetCollisionLayer(v CollisionLayer) { e.world.CollisionLayer.Set(e.id, v) }

func (e Entity2) SetCollisionGeometry(v string) { e.world.CollisionGeometry.Set(e.id, v) }

func (e Entity2) SetPhysicsMassOverride(v float32) { e.world.PhysicsMassOverride.Set(e.id, v) }

// TODO: this is 1) incorrect 2) should probably be generalized somehow
func (e Entity2) SetShouldSetOffFuseOnImpact(v bool) { e.world.ShouldSetOffFuseOnImpact.Set(e.id, struct{}{}) }

func (e Entity2) Velocity() Velocity { v, _ := e.world.Velocity.Get(e.id); return v }

func (e Entity2) SetVelocity(v Velocity) { e.world.Velocity.Set(e.id, v) }

func (e Entity2) SetVisibilityCondition(v VisibilityCondition) { e.world.VisibilityCondition.Set(e.id, v) }

func (e Entity2) SetCosmeticOffset(v CosmeticOffset) { e.world.CosmeticOffset.Set(e.id, v) }

func (e Entity2) SetRenderingGeometry(v string) { e.world.RenderingGeometry.Set(e.id, v) }

func (e Entity2) SetSoundEffect(v SoundEmitter) { e.world.SoundEffect.Set(e.id, v) }

func (e Entity2) MarkForDeletion() { e.world.Delete.Set(e.id, struct{}{}) }
