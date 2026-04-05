package game

import (
	"io/fs"
	"log/slog"
	"reflect"
	"time"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/physics"
)

// TODO: actually indeed stick it onto Scene or pass it through UpdateInfo
var Data fs.FS

// TODO: use "object" instead of "entity" throughout the code?

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

// TODO: introduce Camera component which will specify fov etc
type Camera struct {
	FieldOfView float32
}

// TODO: fold almost all scripty stuff into the same column? Although I guess
// that's more up to how we wanna represent things for authoring etc. I'm not
// sure it would make sense to fold animation graph script into the same column
// as game logic script. However...

// TODO: add a type for representing non-mutable lists and stuff?

type Columns struct {
	// Name ecs.ComponentStore[string]

	// TODO: explore if we can make CreateEntity set this component
	//CreationTime ecs.Column[Time]

	// Do not access this column directly; use {Get,Set}Parent. Specifies the
	// parent object.
	Parent ecs.Column[ecs.ID]
	// The bone in the parent's skeleton that transforms this object.
	ParentBone ecs.Column[string]
	// Do not access this column directly; use {Get,Set}Transform and
	// GetGlobalTransform instead.
	//
	// The translation and rotation parts of the object's parent-relative
	// transform.
	TransformTR ecs.Column[TR3f64]
	// Do not access this column directly; use {Get,Set}Transform and
	// GetGlobalTransform instead.
	//
	// The scale and shearing part of the object's parent-relative transform.
	//
	// It is possible for there to be an entry in TransformTR but not in
	// TransformS, in which case no scaling or shearing is applied to the
	// object.
	TransformS ecs.Column[gmath.Mat3x3Uf32]

	Skeleton ecs.Column[string]

	// Testing only. We'll make this a shadow column which will be updated by an
	// animation graph esque thingy. Also Armature should probably be a separate
	// column so that bone parenting works even when there's no animation
	// playing... On the other hand idk actually there's not really much point
	// to having Armature but no Pose, hmm. I guess it doesn't really matter if
	// we're going to switch to script anyway. Actually it does: having a
	// separate Armature column would let us have no animation script assigned.
	//
	// Right now we're also serializing the entire skeleton over the network.
	//
	// TODO: replace this with Animgraph or something column
	Pose ecs.Column[animgraph.Pose]

	// Physics should only run for bodies that have no parent. We could
	// generalize a little by having an entity be "physics scene" and run sim
	// for the immediate children (but not for the grandchildren), basically.
	// Though on the other hand it might turn out to be annoying to figure out
	// what to parent newly spawned object to.

	// NOTE: constraints and pairwise filter
	//
	// Should we have an identifier for each filtered/constrained entity so that
	// we can have multiple filters against the same entity? Another option
	// would be to have any entity specify filtered/constrained pairs. Yet
	// another would be to have constraints and nocollide pairs be its own
	// concept in the world

	// TODO: merge some of these components?

	Velocity               ecs.Column[Velocity]
	CollisionGeometry      ecs.Column[string]
	CollisionLayer         ecs.Column[CollisionLayer]
	PhysicsFilter          ecs.Column[[]ecs.ID] // TODO: generalize to all physics constraints
	GravityFactor          ecs.Column[float32]
	PhysicsMassOverride    ecs.Column[float32] // TODO: remove "Physics" prefix from these
	PhysicsInertiaOverride ecs.Column[gmath.Mat4x4f32]

	// TODO: generalize to all events, including damage etc?
	ContactEvents ecs.Column[[]ContactEvent]

	// TODO: kill this column and handle it at prefab instantination
	CollectionInstance ecs.Column[CollectionInstance]

	// Logic

	Timer ecs.Column[Time]

	Entity ecs.Column[Entity] // TODO: rename to Logic or Script

	// Renderer

	VisibilityMask ecs.Column[VisibilityMask]

	CosmeticOffset                ecs.Column[CosmeticOffset]
	DeleteCosmeticOffsetOnContact ecs.Column[struct{}]

	RenderingGeometry ecs.Column[string]

	SoundEffect      ecs.Column[SoundEmitter] // TODO: should be a simple filename string
	SoundEffectState ecs.Column[LoopedSound]

	// Deletion

	// TODO: move it out of the Columns struct, there's no reason to replicate this
	Delete ecs.Column[struct{}]
}

type Scene struct {
	// TODO: could we move this to somewhere? Either way I would prefer if
	// things would consult UpdateParams.Now rather than Scene.Params
	Now Time

	Table *ecs.Table
	Columns

	// TODO: delegate managing this to the caller? We would get access to this
	// with UpdateInfo
	physicsSystem     *physics.System
	physicsBodyExists ecs.Column[struct{}]

	Logger *slog.Logger
}

func NewScene(n int) *Scene {
	w := new(Scene)

	w.Table = ecs.NewTable(n)

	// TODO: make it clear that these are reflect references

	columns := reflect.ValueOf(&w.Columns).Elem()
	for i := range columns.Type().NumField() {
		columns.Field(i).Addr().Interface().(interface{ Init(*ecs.Table) }).Init(w.Table)
	}

	w.physicsSystem = physics.NewSystem(
		int(NumBroadPhaseLayers),
		int(NumPhysicsLayers),
		PhysicsLayerToBroadPhaseLayer[:],
		ShouldPhysicsLayersCollide)
	w.physicsBodyExists.Init(w.Table)

	w.Logger = slog.Default()

	// TODO: we should expose an OptimizeBroadPhase call on physicsSystem which
	// we'll (optionally) call after loading the world and perhaps every so
	// often

	return w
}

func (w *Scene) Destroy() {
	// TODO: stop and destroy physicsSystem here
}

// TODO: rename to EntityExists
func (w *Scene) IsEntityValid(id ecs.ID) bool { return w.Table.IDs().Exists(id) }

// TODO: do we need client-only entities? I don't think we do with this tbh
// TODO: make this private?
// TODO: same as DeleteEntityImmediately, we probably should make a column for
// creating entities so we can run a processing pass in parallel that then
// spawns entities. Except this column would have to be of funcs.
func (w *Scene) CreateEntity(info *UpdateParams) ecs.ID {
	if info.Speculating {
		// Create an entity at high index and mark it speculative so that it
		// gets removed when we receive the update for this tick.
		panic("not implemented")
	}
	return w.Table.Alloc()
}

// This is used by client networking to remove entities.
//
// TODO: is the way we use it correct (deleting entities in-between ticks?)
// TODO: could we bulk delete things?
// TODO: kill in favor of w.Table.Delete(id) doing the expected thing.
func (w *Scene) DeleteEntityImmediately(id ecs.ID) {
	// TODO: we should just make a physicsBodyColumn or something
	if _, ok := w.physicsBodyExists.Get(id); ok {
		w.physicsSystem.RemoveBody(physics.BodyID(id))
	}
	w.Table.Delete(id)
}

func (w *Scene) Globals() SceneGlobals {
	globals, _ := SceneGetEntity[SceneGlobals](w, 1)
	return globals
}

func (scene *Scene) GetParent(id ecs.ID) ecs.ID {
	parent, _ := scene.Parent.Get(id)
	return parent
}

func (scene *Scene) SetParent(id, parent ecs.ID) {
	if parent != 0 {
		scene.Parent.Set(id, parent)
	} else {
		// TODO: our panic behavior should be the same as if we tried to Set
		// this id. I.e. if this id isn't valid, we should crash.
		scene.Parent.Delete(id)
	}

	// children, _ := scene.Children.Load(parent)
	// children
}

func (scene *Scene) GetTransform(id ecs.ID) gmath.TRS3f64 {
	tr, ok := scene.TransformTR.Get(id)
	if !ok {
		scene.Logger.Warn("transform queried on an object that has none", "id", id)
		return gmath.TRS3One[float64]()
	}
	s, ok := scene.TransformS.Get(id)
	if !ok {
		s = gmath.Mat3x3UOne[float32]()
	}
	return gmath.TRS3f64{tr.T, tr.R, s}
}

func (scene *Scene) SetTransform(id ecs.ID, T gmath.TRS3f64) {
	scene.TransformTR.Set(id, TR3f64{T.T, T.R})
	if T.S != gmath.Mat3x3UOne[float32]() {
		scene.TransformS.Set(id, T.S)
	} else {
		scene.TransformS.Delete(id)
	}
}

// TODO: if we encounter errors during hierarchy traversal we should restart
// traversal with diagnostics collection and print the collected diagnostics
// after.
//
// TODO: cycle detection
//
// TODO: replace T.Mul(A) with just A on the first iteration to optimize the
// common case
//
// TODO: clean up
func (scene *Scene) GetGlobalTransform(id ecs.ID) gmath.Affine3f64 {
	getTransform := func(id ecs.ID) gmath.Affine3f64 {
		tr, ok := scene.TransformTR.Get(id)
		if !ok {
			return gmath.Affine3One[float64]()
		}
		s, ok := scene.TransformS.Get(id)
		if !ok {
			s = gmath.Mat3x3UOne[float32]()
		}
		return gmath.TRS3f64{tr.T, tr.R, s}.Compose()
	}

	getBoneTransform := func(id ecs.ID, bone string) gmath.Affine3f32 {
		pose, ok := scene.Pose.Get(id)
		if !ok {
			return gmath.Affine3One[float32]()
		}
		skelly := scene.GetSkeleton(id)
		if skelly == nil {
			return gmath.Affine3One[float32]()
		}
		boneIndex := skelly.JointByName(bone)
		if boneIndex == -1 {
			return gmath.Affine3One[float32]()
		}
		boneTransform, ok := pose.Bones[boneIndex]
		if !ok {
			return skelly.BindPose[boneIndex]
		}
		return boneTransform.Mul(skelly.BindPose[boneIndex])
	}

	// TODO: don't hardcode the hierarchy depth bound
	//
	// NOTE: the hierarchy depth is bounded by no. of entries in the table

	A := gmath.Affine3One[float64]()
	for range 5000 {
		A = getTransform(id).Mul(A)

		parent := scene.GetParent(id)
		if parent == 0 {
			// TODO: ensure that parent to bone isn't set
			break
		}

		if parentBone, parentedToBone := scene.ParentBone.Get(id); parentedToBone {
			A = gmath.Affine3Convert[float64](getBoneTransform(parent, parentBone)).Mul(A)
		}

		id = parent
	}

	return A
}

func (scene *Scene) GetSkeleton(id ecs.ID) *animgraph.Skeleton {
	skellyName, ok := scene.Skeleton.Get(id)
	if !ok {
		return nil
	}
	return skeleton(skellyName)
}

// TODO: convert this to generic method once generic methods land
func SceneGetEntity[T Entity](w *Scene, id ecs.ID) (T, bool) {
	entity, _ := w.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
}

type UpdateParams struct {
	// Now         Time // for substeps
	Δt          time.Duration
	Speculating bool
	Logger      *slog.Logger // TODO: kill this in favor of scene.Logger
}

// TODO: parallel for in blender for example specifies bulk number for tasks so
// we might want to do the same.

func (w *Scene) Step(updateParams *UpdateParams) {
	w.Now = w.Now.Add(updateParams.Δt)

	w.processTimers(updateParams)

	// TODO: optimize loops over entities implementing particular interface by
	// having shadow columns.

	for id, entity := range ecs.All(&w.Entity) {
		if player, ok := entity.(Player); ok {
			player.PlayerUpdate(w, id, updateParams)
		}
	}

	// TODO: move into updatePhysicsShadow?
	for id, entity := range ecs.All(&w.Entity) {
		if entity, ok := entity.(UpdateBeforePhysics); ok {
			entity.UpdateBeforePhysics(w, id, updateParams)
		}
	}

	w.updatePhysicsShadow()
	w.physicsStep(updateParams.Δt)

	for id, entity := range ecs.All(&w.Entity) {
		if entity, ok := entity.(UpdateAfterPhysics); ok {
			entity.UpdateAfterPhysics(w, id, updateParams)
		}
	}

	for id, v := range ecs.Join(&w.ContactEvents, &w.DeleteCosmeticOffsetOnContact) {
		for _, ce := range v.V1 {
			if ce.Type == 1 {
				w.CosmeticOffset.Delete(id)
				break
			}
		}
	}

	for id, a := range ecs.All(&w.SoundEffectState) {
		soundEffect, _ := w.SoundEffect.Get(id)
		if soundEffect.PlayTime.Add(time.Duration(a.LengthInSamples * 1e9 / 48000)).After(w.Now) {
			continue
		}

		soundEffect.Effect = a.Sound
		soundEffect.Attenuation = a.Attenuation
		soundEffect.PlayTime = w.Now
		w.SoundEffect.Set(id, soundEffect)
	}

	// TODO: move this to happen earlier
	w.DeleteEntities()

	// TODO: move this to happen earlier
	w.ContactEvents.Clear()
}

func mustOk[T any](v T, ok bool) T {
	if !ok {
		panic("not ok")
	}
	return v
}

// TODO: rename to make it clear that we're deleting things already marked for
// deletion.
func (w *Scene) DeleteEntities() {
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

			if _, delet := w.Delete.Get(id); delet {
				return true
			}

			delet := f(w.GetParent(id))
			if delet {
				w.Delete.Set(id, struct{}{})
			}
			return delet
		}

		for id := range ecs.All(&w.Parent) {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	{
		// TODO: delete entities in bulk?
		for id := range ecs.All(&w.Delete) {
			w.DeleteEntityImmediately(id)
		}

		for range ecs.All(&w.Delete) {
			panic("all columns must be empty")
		}
	}
}
