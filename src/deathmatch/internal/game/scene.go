package game

import (
	"fmt"
	"io/fs"
	"log/slog"
	"reflect"
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
	"worldspawn/physics"
)

var Data fs.FS

// TODO: use "object" instead of "entity" throughout the code?

// TODO: split this file up

type TranslationRotation struct {
	Translation geometry.DVec3
	Rotation    geometry.Rot3
}

type Velocity struct {
	Linear  geometry.Vec3
	Angular geometry.Vec3
}

type SceneGlobals struct {
	// TODO: replace it with sky material
	Sky string

	Gravity geometry.Vec3
}

func (SceneGlobals) entity() {}

// TODO: introduce Camera component which will specify fov etc
type Camera struct {
	FieldOfView float32
}

// TODO: add a type for representing non-mutable lists and stuff?

// TODO: we can update OldWorld (against which we compute delta) only sometimes
// and incrementally (this is the actually useful bit, as updating "sometimes"
// will make certain ticks longer, potentially making them miss deadline.) we
// should also update it reactively when we add dirty tracking, to minimize the
// amount of work we do.

// TODO: introduce struct tags like compatibility names etc
type Columns struct {
	// Name ecs.ComponentStore[string]

	// TODO: explore if we can make CreateEntity set this component
	CreationTime ecs.Column[Time]

	Parent ecs.Column[ecs.ID]
	// Children ecs.ComponentStore[map[ecs.ID] // map[ecs.ID]struct{} ?

	// Do not access this column directly, use {Get,Set}{Local,Global}TRS
	// instead.
	LocalTranslationRotation ecs.Column[TranslationRotation]
	// Do not access this column directly, use {Get,Set}{Local,Global}TRS
	// instead.
	LocalScale ecs.Column[geometry.Vec3]

	Velocity ecs.Column[Velocity]

	// NOTE: constraints and pairwise filter
	//
	// Should we have an identifier for each filtered/constrained entity so that
	// we can have multiple filters against the same entity? Another option
	// would be to have any entity specify filtered/constrained pairs. Yet
	// another would be to have constraints and nocollide pairs be its own
	// concept in the world

	// TODO: merge some of these components?

	CollisionGeometry      ecs.Column[string]
	CollisionLayer         ecs.Column[CollisionLayer]
	PhysicsFilter          ecs.Column[[]ecs.ID] // TODO: generalize to all physics constraints
	GravityFactor          ecs.Column[float32]
	PhysicsMassOverride    ecs.Column[float32] // TODO: remove "Physics" prefix from these
	PhysicsInertiaOverride ecs.Column[geometry.Mat4x4]

	PlayerSpawn ecs.Column[struct{}]

	DeleteAfter ecs.Column[Time]

	// Timer ecs.ComponentStore[time.Duration]

	// TODO: generalize to all events, including damage etc?
	ContactEvents ecs.Column[[]ContactEvent]

	// TODO: rename to just Collection?
	CollectionInstance ecs.Column[CollectionInstance]

	// TODO: rename, to e.g. Logic? Or Any?
	Entity ecs.Column[Entity]

	Delete ecs.Column[struct{}]

	CosmeticOffset                ecs.Column[CosmeticOffset]
	DeleteCosmeticOffsetOnContact ecs.Column[struct{}]

	Visibility ecs.Column[Visibility]

	RenderingGeometry ecs.Column[string]

	// TODO: rename to SoundEmitter
	SoundEffect ecs.Column[SoundEmitter]
}

type Scene struct {
	Now Time

	Table *ecs.Table
	Columns
	physicsSystem     *physics.System
	physicsBodyExists ecs.Column[struct{}]
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
		scene.Parent.Delete(id)
	}

	// children, _ := scene.Children.Load(parent)
	// children
}

func (scene *Scene) GetLocalTRS(id ecs.ID) (geometry.DTRS3, bool) {
	tr, ok := scene.LocalTranslationRotation.Get(id)
	if !ok {
		return geometry.DTRS3One(), false
	}
	s, ok := scene.LocalScale.Get(id)
	if !ok {
		s = geometry.Vec3Broadcast(1)
	}
	return geometry.DTRS3{tr.Translation, tr.Rotation, s}, true
}

// TODO: should we blow up if we have no parent? That doesn't seem very sensible TBH.
func (scene *Scene) SetLocalTRS(id ecs.ID, trs geometry.DTRS3) {
	scene.LocalTranslationRotation.Set(id, TranslationRotation{trs.T, trs.R})
	// TODO: scale
}

func (scene *Scene) GetGlobalTRS(id ecs.ID) (geometry.DTRS3, bool) {
	result := geometry.DTRS3One()
	// TODO: return false for id == 0
	for id != 0 {
		trs, ok := scene.GetLocalTRS(id)
		if !ok {
			return geometry.DTRS3One(), false
		}
		result = trs.Mul(result)
		id = scene.GetParent(id)
	}
	return result, true
}

func (scene *Scene) SetGlobalTRS(id ecs.ID, trs geometry.DTRS3) {
	// TODO: handle having a parent in some way
	scene.SetLocalTRS(id, trs)
}

func SceneGetEntity[T any](w *Scene, id ecs.ID) (T, bool) {
	entity, _ := w.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
}

// TODO: move these into a separate file

// TODO: split stuff relevant for entity creation into its own type pointing to
// this object (entity creation needs to be aware of Now and Speculating and be
// able to log things also.)
type UpdateParams struct {
	// Now         Time // for substeps
	Δt          time.Duration
	Speculating bool
	Logger      *slog.Logger
}

func (w *Scene) HandleInput(id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	if player, ok := SceneGetEntity[Player](w, id); ok {
		player.PlayerSubstep(w, id, cmd, info)
	} else {
		info.Logger.Warn(fmt.Sprintf("entity does not exist or is not %s", reflect.TypeFor[Player]().Name()), "id", id)
	}
}

// TODO: parallel for in blender for example specifies bulk number for tasks so
// we might want to do the same.

func (w *Scene) Step(updateParams *UpdateParams) {
	w.Now = w.Now.Add(updateParams.Δt)

	// TODO: optimize loops over entities implementing particular interface by
	// having shadow component stores.

	for id, entity := range ecs.All(&w.Entity) {
		if player, ok := entity.(Player); ok {
			player.PlayerUpdate(w, id, updateParams)
		}
	}

	for id, entity := range ecs.All(&w.Entity) {
		if entity, ok := entity.(UpdateBeforePhysics); ok {
			entity.UpdateBeforePhysics(w, id, updateParams)
		}
	}

	w.worldToPhysics()
	w.physicsStep(updateParams.Δt)

	for id, entity := range ecs.All(&w.Entity) {
		if entity, ok := entity.(UpdateAfterPhysics); ok {
			entity.UpdateAfterPhysics(w, id, updateParams)
		}
	}

	/*
		ecs.Range(
			w.ContactEvents,
			func(entityID ecs.ID, contactEvents []ContactEvent) {
				for _, ce := range contactEvents {
					switch ce.Type {
					case 1:
						w.SoundEffect.Store(entityID, SoundEffect{
							Effect:   "grenade_launcher_fire.wav",
							PlayTime: w.Time + Δt + time.Duration(rand.Int63n(int64(10*time.Millisecond))),
						})
					}
				}
			})
	*/

	// for

	// TODO: we want a unique primary component here, something with Grenade in
	// the name perhaps
	/*
		ecs.Range2(
			w.ContactEvents, w.Explosive,
			func(entityID ecs.ID, contactEvents []ContactEvent, explosive DamageAndKnockback) {
				for _, ce := range contactEvents {
					switch ce.Type {
					case 1:
						// BUG: move the following to its own system

						positionRotation, _ := w.PositionRotation.Load(entityID)

						ecs.Range(
							w.PhysicsMotionType,
							func(entityID2 ecs.ID, motionType PhysicsMotionType) {
								// Should we only apply this stuff to dynamic
								// objects, and kinematic objects should be opted in
								// on per object basis? Or should objects be opted
								// out of knockback? For example we don't want
								// payload cart to get any knockback.
								if motionType == PhysicsMotionStatic {
									return
								}
								if entityID == entityID2 {
									return
								}

								positionRotation2, _ := w.PositionRotation.Load(entityID2)
								velocity, _ := w.Velocity.Load(entityID2)

								// TODO: I'm really not sure how to best gauge
								// distance and what would be a good way to
								// implement explosions
								hello := positionRotation2.Position.Add(geometry.DVec3{Z: 0.8}).Sub(positionRotation.Position).Vec3()

								a, _ := w.DistanceBasedImpactMultiplierTable.Load(entityID)

								len := hello.Length()

								var impactMultiplier float32
								for _, e := range a {
									if len >= e.Distance {
										impactMultiplier = e.Multiplier
									}
								}

								if impactMultiplier > 0 {
									// TODO: pick a direction somehow if len == 0
									dir := hello.Scale(1.0 / len)

									force := dir.Scale(explosive.Knockback * impactMultiplier)

									// TODO: make it depend on the mass and other things probably?

									velocity.Linear = velocity.Linear.Add(force)
									w.Velocity.Store(entityID2, velocity)
								}
							})

						// Schedule this entity for removal
						w.Remove.Store(entity, struct{}{})

						// TODO: we could reuse the current entity for an effect. Should we?

						effect := w.AllocEntityID()
						w.PositionRotation.Store(effect, positionRotation)
						w.SoundEffect.Store(effect, SoundEffect{
							Effect:   "later.wav",
							PlayTime: w.Time + Δt,
						})
						// TODO: remove affect after a while

						return
					}
				}
			})
	*/

	for id, v := range ecs.Join(&w.ContactEvents, &w.DeleteCosmeticOffsetOnContact) {
		for _, ce := range v.V1 {
			if ce.Type == 1 {
				w.CosmeticOffset.Delete(id)
				break
			}
		}
	}

	for id, deleteAfter := range ecs.All(&w.DeleteAfter) {
		if deleteAfter.Before(w.Now) {
			w.Delete.Set(id, struct{}{})
		}
	}

	w.DeleteEntities()
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
		for id := range ecs.All(&w.Delete) {
			// TODO: delete bodies in bulk
			if _, ok := w.physicsBodyExists.Get(id); ok {
				w.physicsSystem.RemoveBody(physics.BodyID(id))
			}
			w.Table.Delete(id)
		}

		for range ecs.All(&w.Delete) {
			panic("all columns must be empty")
		}
	}
}

// TODO: fold into Update
func ClearTransientComponents(w *Scene) {
	w.ContactEvents.Clear()
}
