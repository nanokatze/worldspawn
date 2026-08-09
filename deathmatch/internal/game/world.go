package game

import (
	"io/fs"
	"log/slog"
	"reflect"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/ecs/bitset"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
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

	ScriptState ecs.Column[ScriptState]

	// TODO: this will need to be a wheel rather than a plain column
	NextThink ecs.Column[Time]

	// TODO: require that entities that we're do collision/physics for have no
	// parent. Or maybe allow that somehow, e.g. by having children be joined
	// with the collision geometry of the parent?

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
	Skeleton ecs.Column[unique.Handle[string]]

	pose []skeleton.Pose `worldspawn:"dontreplicate"`

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
	CollectionInstance ecs.Column[CollectionInstance] `worldspawn:"dontreplicate"`

	// Other gameplay stuff

	// Whether the fuse of certain projectiles should be set off when they
	// collide with this entity.
	ShouldSetOffFuseOnImpact ecs.Column[struct{}]

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
	Speculation ecs.Column[Time] `worldspawn:"dontreplicate"`

	// TODO: wrap this (along with physicsSystem) into a type that implements Column interface.
	physics           *physics.System      `worldspawn:"dontreplicate"`
	physicsBodyExists ecs.Column[struct{}] `worldspawn:"dontreplicate"`

	// Entities marked for deletion
	delete bitset.Bitset `worldspawn:"dontreplicate"`
}

// TODO: make all of the internals private. We'll need to add infrastructure for
// replication to World first.
// TODO: I'm really tempted to stick logger onto World atp
type World struct {
	// TODO: don't use this, use info.Now instead
	Now Time

	NextID            ecs.ID
	NextIDSpeculative ecs.ID

	Table *ecs.Table
	// TODO: stop embedding this and make it private
	Columns

	entityUpdates, entityUpdates2 [][]updatef
	globalUpdates                 []func(*UpdateParams, *World)

	logger *slog.Logger
}

func (world *World) Reset(n int) {
	world.Table = ecs.NewTable(n)

	columns := reflect.ValueOf(&world.Columns).Elem()
	for i := range columns.Type().NumField() {
		if !columns.Type().Field(i).IsExported() {
			continue
		}
		col, ok := columns.Field(i).Addr().Interface().(interface{ Init(*ecs.Table) })
		if ok {
			col.Init(world.Table)
		}
	}

	world.Columns.pose = make([]skeleton.Pose, n)

	world.entityUpdates = make([][]updatef, n)
	world.entityUpdates2 = make([][]updatef, n)

	// TODO: we could expose an OptimizeBroadPhase call on physicsSystem I suppose.
	world.Columns.physics = physics.NewSystem(
		int(numCollisionLayers),
		collisionLayerToBroadPhaseLayer[:],
		collisionLayerRules)
	world.Columns.physicsBodyExists.Init(world.Table)

	world.Columns.delete = bitset.Make(n)

	// TODO: the user should pass this
	world.logger = slog.Default()
}

// func (world *World) SetLogger(logger *slog.Logger) { world.logger = logger }

func (world *World) Cap() int { return world.Table.IDs().Cap() }

// TODO: make this private?
// TODO: kill this in favor of IO.Create
func (world *World) CreateEntity(info *UpdateParams) Entity {
	// TODO: don't hardcode index ranges

	if info.Speculating {
		id := world.Table.CreateRowAuto(900, 999, &world.NextIDSpeculative)
		world.Speculation.Set(id, world.Now)
		return Entity{world, id}
	}

	id := world.Table.CreateRowAuto(1, 899, &world.NextID)
	return Entity{world, id}
}

// This is used by client networking to remove entities.
func (world *World) DeleteEntityImmediately(id ecs.ID) {
	world.Columns.pose[id.Index()] = skeleton.Pose{}
	if _, ok := world.physicsBodyExists.Get(id); ok {
		world.physics.RemoveBody(physics.BodyID(id.Index()))
	}
	world.Table.DeleteRow(id)
}

// TODO: kill all of these helpers

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

func (world *World) GetSkeleton(id ecs.ID) *skeleton.Skeleton {
	skellyName, _ := world.Skeleton.Get(id)
	if skellyName.Value() == "" {
		return nil
	}
	return skeletonCache.Get(skellyName)
}

// TODO: https://github.com/nanokatze/worldspawn-deathmatch/issues/68
func (world *World) Entity(id ecs.ID) Entity {
	if !world.Table.IDs().Exists(id) {
		return Entity{}
	}
	return Entity{world, id}
}
