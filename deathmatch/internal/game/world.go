package game

import (
	"fmt"
	"io/fs"
	"log/slog"
	"reflect"
	"time"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: actually indeed stick it onto Scene or pass it through UpdateInfo
var Data fs.FS

// TODO: split this file up

type SceneGlobals struct {
	// TODO: replace it with sky material
	Sky string

	// TODO: create a separate "physics world" entity/component and move this
	// stuff there
	Gravity gmath.Vec3f32
}

func (SceneGlobals) entity() {}

type TR3f64 struct {
	T gmath.Vec3f64
	R gmath.Rot3
}

// TODO: introduce Camera component which will specify fov etc. I guess we could
// also just use Entity.
type Camera struct {
	FieldOfView float32
}

// TODO: fold almost all scripty stuff into the same column? Although I guess
// that's more up to how we wanna represent things for authoring etc. I'm not
// sure it would make sense to fold animation graph script into the same column
// as game logic script. However...

// TODO: add a type for representing non-mutable lists and stuff?

// TODO: certain columns could've perfectly feasibly been unique.Handle[string].
// E.g. Script, CollisionGeometry and RenderingGeometry.
type Columns struct {
	// Name ecs.ComponentStore[string]

	// Logic

	Script ecs.Column[string]
	// TODO: rename to ScriptState
	Entity ecs.Column[Entity]

	NextThink ecs.Column[Time]

	// TODO: this doesn't really need to be a column, this could perfectly
	// feasibly be a plain array with a bitmap.
	Updates ecs.Column[[]func(world *World, id ecs.ID, updateParams *UpdateParams)] `worldspawn:"transient"`

	// Entities marked for deletion
	//
	// TODO: we could try doing immediate deletion or at least fold processing
	// of deletion into process updates?
	Delete ecs.Column[struct{}] `worldspawn:"transient"`

	// Do not access this column directly; use {Get,Set}Parent. Specifies the
	// parent entity.
	Parent ecs.Column[ecs.ID]
	// The bone in the parent's skeleton that transforms this entity.
	ParentBone ecs.Column[string]
	// Do not access this column directly; use {Get,Set}Transform and
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
	// It is possible for there to be an entry in TransformTR but not in
	// TransformS, in which case no scaling or shearing is applied to the
	// entity.
	TransformS ecs.Column[gmath.Mat3x3Uf32]

	// TODO: make skeletons part of geometry?
	Skeleton ecs.Column[string]

	// TODO: this should just point to an animgraph script
	Pose ecs.Column[animgraph.Pose]

	// TODO: require that entities that we're do collision/physics for have no
	// parent. Or maybe allow that somehow, e.g. by having children be joined
	// with the collision geometry of the parent?

	// Collision

	CollisionLayer    ecs.Column[CollisionLayer]
	CollisionGeometry ecs.Column[string]

	Sensor ecs.Column[struct{}]

	// Motion

	// Motion ecs.Column[MotionType]
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

	CosmeticOffset                ecs.Column[CosmeticOffset]
	DeleteCosmeticOffsetOnContact ecs.Column[struct{}]

	VisibilityMask ecs.Column[VisibilityMask]

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

	// Speculative object deadline
	// TODO: rename
	Speculation ecs.Column[Time] // TODO: do not network this
}

// TODO: make all of the internals. We'll need to add infrastructure for
// replication to World first.
type World struct {
	// TODO: could we move this to somewhere? Either way I would prefer if
	// things would consult UpdateParams.Now rather than Scene.Params
	Now Time

	NextID            ecs.ID
	NextIDSpeculative ecs.ID

	Table *ecs.Table
	Columns

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

// TODO: replace CreateEntity and DeleteEntityImmediately with
// EnqueueCreateEntity(f func(id)) and EnqueueDeleteEntity(id)?

// TODO: make this private?
// TODO: same as DeleteEntityImmediately, we probably should make a column for
// creating entities so we can run a processing pass in parallel that then
// spawns entities. Except this column would have to be of funcs.
func (world *World) CreateEntity(info *UpdateParams) ecs.ID {
	// TODO: don't hardcode index ranges

	if info.Speculating {
		id := world.Table.CreateRowAuto(900, 999, &world.NextIDSpeculative)
		world.Speculation.Set(id, world.Now)
		return id
	}

	id := world.Table.CreateRowAuto(1, 899, &world.NextID)
	return id
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

func (world *World) Globals() SceneGlobals {
	globals, _ := SceneGetEntity[SceneGlobals](world, 1)
	return globals
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
			A = gmath.Affine3Convert[float64](getBoneTransform(parent, parentBone)).Mul(A)
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

// TODO: kill this
func SceneGetEntity[T Entity](world *World, id ecs.ID) (T, bool) {
	entity, _ := world.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
}

// TODO: rename to StepContext or something
type UpdateParams struct {
	// Now         Time // for substeps
	Δt          time.Duration
	Speculating bool
	Logger      *slog.Logger
}

// TODO: kill
type Thinker interface {
	Entity

	Think(world *World, id ecs.ID, updateParams *UpdateParams)
}

func (world *World) EnqueueEntityUpdate(to ecs.ID, f func(world *World, id ecs.ID, updateParams *UpdateParams)) {
	updates, _ := world.Updates.Get(to)
	world.Updates.Set(to, append(updates, f))
}

func (world *World) processEntityUpdates(updateParams *UpdateParams) {
	// TODO: double buffer messages so that messages can be sent during
	// processing and process them until there's no more messages.

	for id, updates := range ecs.All(&world.Updates) {
		for _, f := range updates {
			f(world, id, updateParams)
		}
	}

	world.Updates.Clear()

	world.deleteMarkedEntities()
}

func (world *World) think(updateParams *UpdateParams) {
	// TODO: update systems which are allowed to be queried from Think
	// w.updatePhysicsShadow(updateParams)

	for id, scriptName := range ecs.All(&world.Script) {
		script := scripts[scriptName]
		if script.Think == nil {
			continue
		}

		// TODO: we'll want a timer wheel of sorts to make this fast
		nextThink, _ := world.NextThink.Get(id)
		if world.Now.Before(nextThink) {
			continue
		}

		script.Think(world, id, updateParams)
	}

	// TODO: kill this
	for id, entity := range ecs.All(&world.Entity) {
		thinker, ok := entity.(Thinker)
		if !ok {
			continue
		}

		// TODO: we'll want a timer wheel of sorts to make this fast
		nextThink, _ := world.NextThink.Get(id)
		if world.Now.Before(nextThink) {
			continue
		}

		thinker.Think(world, id, updateParams)
	}

	world.processEntityUpdates(updateParams)
}

// TODO: rename to make it clear that we're deleting things already marked for
// deletion.
func (world *World) deleteMarkedEntities() {
	// TODO: would we benefit from (optionally) checking whether any entities
	// have dangling references to other entities?

	// TODO: delete entities too far off the map

	// Propagate deletion from parents.
	//
	// TODO: make this less gross. We could do a probe whether there's any
	// deletions at all right now.
	{
		var f func(id ecs.ID) bool
		// TODO: we could also rotate this
		f = func(id ecs.ID) bool {
			if id == 0 {
				return false
			}

			if _, delet := world.Delete.Get(id); delet {
				return true
			}

			delet := f(world.GetParent(id))
			if delet {
				world.Delete.Set(id, struct{}{})
			}
			return delet
		}

		for id := range ecs.All(&world.Parent) {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	for id := range ecs.All(&world.Delete) {
		if _, ok := world.physicsBodyExists.Get(id); ok {
			world.physics.RemoveBody(physics.BodyID(id.Index()))
		}
		world.Table.DeleteRow(id)
	}
}
