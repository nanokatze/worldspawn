package game

import (
	"io/fs"
	"reflect"
	"unique"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/ecs"
	"worldspawn/internal/ecs/bitset"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: stick it into World or pass it through UpdateInfo
var Data fs.FS

// TODO: fold almost all scripty stuff into the same column? Although I guess
// that's more up to how we wanna represent things for authoring etc. I'm not
// sure it would make sense to fold animation graph script into the same column
// as game logic script. However...

// TODO: add a type for representing non-mutable lists and stuff?

// TODO: instead of ecs.Columns, these should be any objects implementing an
// interface with Get/Set/iter. We'll eventually replace ecs.Column with simpler
// structures keyed by plain integer indices and get rid of ecs.Table, bringing
// entity ID validation into GetEntity2/3 into.

type Columns struct {
	// Entity programmability

	Name ecs.Column[unique.Handle[string]]

	// TODO: rename this pls
	Entity ecs.Column[Entity]

	NextThink ecs.Column[Time]

	// TODO: require that entities that we're do collision/physics for have no
	// parent. Or maybe allow that somehow, e.g. by having children be joined
	// with the collision geometry of the parent?

	// Other gameplay stuff

	// Whether the fuse of certain projectiles should be set off when they
	// collide with this entity.
	// TODO: could we generalize this?
	ShouldSetOffFuseOnImpact ecs.Column[struct{}]

	// Transform

	// Do not access this column directly; use {Get,Set}Parent. Specifies the
	// parent entity.
	Parent ecs.Column[ecs.ID]
	// The bone in the parent's skeleton that transforms this entity.
	ParentBone ecs.Column[unique.Handle[string]]
	// Do not access this column directly; use {Get,Set}Transform or
	// GetGlobalTransform instead.
	//
	// The translation and rotation parts of the entity's parent-relative
	// transform.
	// TODO: rename to TranslationAndRotation
	TransformTR ecs.Column[TR3f64]
	// Do not access this column directly; use {Get,Set}Transform and
	// GetGlobalTransform instead.
	//
	// The scale and shearing part of the entity's parent-relative transform.
	//
	// It is possible for an entity to have a TransformTR but not TransformS, in
	// which case no scaling or shearing is applied to the entity.
	// TODO: we'll later hard-require TransformS to always be set to something valid.
	// TODO: rename to ScalingAndShearing
	TransformS ecs.Column[gmath.Mat3x3Uf32]

	// TODO: make skeletons part of geometry? Ok actually wait which geometry
	// lol. We'll need to think more about this.
	Skeleton ecs.Column[unique.Handle[string]]

	// Pose ecs.Column[animgraph.Pose]

	// Collision

	CollisionLayer    ecs.Column[CollisionLayer]
	CollisionGeometry ecs.Column[unique.Handle[string]]

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

	// Renderer

	VisibilityCondition ecs.Column[VisibilityCondition]

	CosmeticOffset ecs.Column[CosmeticOffset]

	RenderingGeometry ecs.Column[unique.Handle[string]]

	// TODO: so the way we'll probably go about sounds is by having 2 networked
	// columns and one shadow column. The shadow column will feed the actual
	// audio player, so it should contain basically sound files and t0 (a mix of
	// game time and sample count) and the networked columns be the "audio
	// program" identifier and audio program state. For now we can start with
	// just the shadow column.

	// TODO: rethink sounds
	SoundEffect      ecs.Column[SoundEmitter]
	SoundEffectState ecs.Column[LoopedSound] // TODO: kill this column

	// Deadlines of speculatively spawned entities.
	//
	// TODO: rename
	Speculation ecs.Column[Time]
}

// TODO: make all of the internals private. We'll need to add infrastructure for
// replication to World first.
// TODO: I'm really tempted to stick logger onto World atp
type World struct {
	// TODO: we could move it to be a WorldGlobals field I suppose
	Now Time

	NextID            ecs.ID
	NextIDSpeculative ecs.ID

	Table *ecs.Table
	Columns

	entityUpdates, entityUpdates2 [][]updatef
	globalUpdates                 []func(*UpdateParams, *World)

	Pose ecs.Column[animgraph.Pose]

	physics *physics.System
	// TODO: this should be folded into physicsSystem
	physicsBodyExists ecs.Column[struct{}]

	// Entities marked for deletion
	// TODO: move this into Entities
	delete bitset.Bitset
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

	world.Pose.Init(world.Table)

	// TODO: pass contact listener. Or make it so that the contact listener is
	// passed at call to Update.
	// TODO: we should expose an OptimizeBroadPhase call on physicsSystem which
	// we'll (optionally) call after loading the world and perhaps every so
	// often
	world.physics = physics.NewSystem(
		int(numCollisionLayers),
		collisionLayerToBroadPhaseLayer[:],
		collisionLayerRules)
	world.physicsBodyExists.Init(world.Table)

	world.delete = bitset.Make(n)

	return world
}

func (world *World) Cap() int { return world.Table.IDs().Cap() }

// TODO: make this private?
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

// This is used by client networking to remove entities.
// TODO: kill
func (world *World) DeleteEntityImmediately(id ecs.ID) {
	if _, ok := world.physicsBodyExists.Get(id); ok {
		world.physics.RemoveBody(physics.BodyID(id.Index()))
	}
	world.Table.DeleteRow(id)
}

// TODO: kill all of these helpers

// TODO: rename this
func (world *World) GetEntity[T Entity](id ecs.ID) (T, bool) {
	entity, _ := world.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
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
}

func (world *World) GetSkeleton(id ecs.ID) *animgraph.Skeleton {
	skellyName, ok := world.Skeleton.Get(id)
	if !ok {
		return nil
	}
	return skeleton(skellyName)
}

// TODO: do validation here
// TODO: rename to just Entity when we rename the Entity column to something
// more reasonable.
func (world *World) GetEntity2(id ecs.ID) Entity2 {
	if world.Table.IDs().Exists(id) {
		return Entity2{world, id}
	}
	return Entity2{}
}

// Entity2 must not be stored in any structures and also not passed across
// update lambdas.
//
// TODO: rename to EntityPtr or something along those lines
type Entity2 struct {
	world *World
	id    ecs.ID // TODO: replace with index
}

func (e Entity2) ID() ecs.ID { return e.id }

func (e Entity2) Valid() bool { return e.id != 0 }

// TODO: rename to something nicer
func (e Entity2) Clear() {
	if _, ok := e.world.physicsBodyExists.Get(e.id); ok {
		// We have to do this because ClearRow unsets the bit in
		// physicsBodyExists, so we end up with an orphan physics body.
		e.world.physics.RemoveBody(physics.BodyID(e.id.Index()))
	}
	e.world.Table.ClearRow(e.id)
}

// TODO: replace with an easy to use thingy for checking whether an entity's
// script satisfies some interface, and stuff for calling that interface?
// TODO: return a pointer instead of struct as is?
func (e Entity2) Script() script {
	return Scripts[reflect.TypeOf(e.world.Entity.Load(e.id.Index()))]
}

// TODO: the zoo of script state stuff really kinda pmo

func (e Entity2) ScriptState() Entity { return e.world.Entity.Load(e.id.Index()) }

func (e Entity2) SetScriptState(v Entity) { e.world.Entity.Store(e.id.Index(), v) }

func (e Entity2) UpdateScriptState[T Entity](f func(v *T)) {
	v := e.world.Entity.Load(e.id.Index()).(T)
	f(&v)
	e.world.Entity.Store(e.id.Index(), v)
}

func (e Entity2) SetNextThink(v Time) { e.world.NextThink.Store(e.id.Index(), v) }

// TODO: this is incorrect
func (e Entity2) SetShouldSetOffFuseOnImpact(v bool) {
	e.world.ShouldSetOffFuseOnImpact.Store(e.id.Index(), struct{}{})
}

func (e Entity2) SetParent(v ecs.ID) { e.world.SetParent(e.id, v) }

func (e Entity2) SetParentBone(v unique.Handle[string]) { e.world.ParentBone.Store(e.id.Index(), v) }

func (e Entity2) Transform() gmath.TRS3f64 {
	// TODO: validate that the transform is invertible? We might wanna ban non-invertible transforms

	tr := e.world.TransformTR.Load(e.id.Index())
	s := e.world.TransformS.Load(e.id.Index())
	return gmath.TRS3f64{tr.T, tr.R, s}
}

func (e Entity2) SetTransform(v gmath.TRS3f64) {
	// TODO: validate the transform

	e.world.TransformTR.Store(e.id.Index(), TR3f64{v.T, v.R})
	e.world.TransformS.Store(e.id.Index(), v.S)
}

func (e Entity2) SetTranslationAndRotation(v TR3f64) { e.world.TransformTR.Store(e.id.Index(), v) }

func (e Entity2) SetSkeleton(v unique.Handle[string]) { e.world.Skeleton.Store(e.id.Index(), v) }

func (e Entity2) SetPose(v animgraph.Pose) { e.world.Pose.Store(e.id.Index(), v) }

func (e Entity2) SetCollisionLayer(v CollisionLayer) { e.world.CollisionLayer.Store(e.id.Index(), v) }

func (e Entity2) SetCollisionGeometry(v unique.Handle[string]) {
	e.world.CollisionGeometry.Store(e.id.Index(), v)
}

func (e Entity2) SetPhysicsMassOverride(v float32) { e.world.PhysicsMassOverride.Store(e.id.Index(), v) }

func (e Entity2) Velocity() Velocity { return e.world.Velocity.Load(e.id.Index()) }

func (e Entity2) SetVelocity(v Velocity) { e.world.Velocity.Store(e.id.Index(), v) }

func (e Entity2) SetVisibilityCondition(v VisibilityCondition) {
	e.world.VisibilityCondition.Store(e.id.Index(), v)
}

func (e Entity2) SetCosmeticOffset(v CosmeticOffset) { e.world.CosmeticOffset.Store(e.id.Index(), v) }

func (e Entity2) SetRenderingGeometry(v unique.Handle[string]) {
	e.world.RenderingGeometry.Store(e.id.Index(), v)
}

func (e Entity2) SetSoundEffect(v SoundEmitter) { e.world.SoundEffect.Store(e.id.Index(), v) }

func (e Entity2) MarkForDeletion() { e.world.delete.Store(e.id.Index(), true) }
