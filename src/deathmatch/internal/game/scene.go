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

// TODO: use "object" instead of "entity" throughout the code?

// TODO: split this file up

// TODO: split stuff relevant for entity creation into its own type pointing to
// this object (entity creation needs to be aware of Now and Speculating and be
// able to log things also.)
type UpdateParams struct {
	// Now         Time // for substeps
	Δt          time.Duration
	Speculating bool
	Logger      *slog.Logger
}

// TODO: kill in favor of a function
type EntityCreationContext struct {
	*UpdateParams

	parentTo ecs.ID
}

// TODO: generalize to arbitrary components, i.e. this should just be a lambda
func (ctx EntityCreationContext) WithParent(parent ecs.ID) EntityCreationContext {
	ctx.parentTo = parent
	return ctx
}

/*
type SubtickUpdateParams struct {
	UpdateParams
}
*/

var Data fs.FS

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

	LocalTranslationRotation ecs.Column[TranslationRotation]
	LocalScale               ecs.Column[geometry.Vec3]

	Velocity ecs.Column[Velocity]

	RenderingGeometry ecs.Column[string]

	// TODO: rename to SoundEmitter
	SoundEffect ecs.Column[SoundEmitter]

	Viewmodel2 ecs.Column[Viewmodel2]

	CosmeticOffset                ecs.Column[CosmeticOffset]
	DeleteCosmeticOffsetOnContact ecs.Column[struct{}]

	// Posing test
	Animation ecs.Column[Animation]

	// TODO: going forward, pose needs to be string -> TRS3. Pose generally will
	// be set in two of the following ways: with physics (so there will be
	// constrained entities representing limbs and we will set the respective
	// joints to where those entities are) and with animation (so there might be
	// a complex script driving the animation)
	Pose ecs.Column[map[string]geometry.Mat4x4]

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

	// TODO: unify these two components probably
	ViewPunch         ecs.Column[geometry.Rot3]
	ViewPunchVelocity ecs.Column[geometry.Vec3]

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
}

type Scene struct {
	Now Time

	IDs *ecs.IDs
	Columns
	physicsSystem     *physics.System
	physicsBodyExists ecs.Column[struct{}]
}

func NewScene(n int) *Scene {
	w := new(Scene)

	w.IDs = ecs.NewEntities(n)

	// TODO: make it clear that these are reflect references

	columns := reflect.ValueOf(&w.Columns).Elem()
	for i := range columns.Type().NumField() {
		columns.Field(i).Addr().Interface().(interface{ Init(*ecs.IDs) }).Init(w.IDs)
	}

	w.physicsSystem = physics.NewSystem(
		int(NumBroadPhaseLayers),
		int(NumPhysicsLayers),
		PhysicsLayerToBroadPhaseLayer[:],
		ShouldPhysicsLayersCollide)
	w.physicsBodyExists.Init(w.IDs)

	// TODO: we should expose an OptimizeBroadPhase call on physicsSystem which
	// we'll (optionally) call after loading the world and perhaps every so
	// often

	return w
}

func (w *Scene) Destroy() {
	// TODO: stop and destroy physicsSystem here
}

// TODO: rename to EntityExists
func (w *Scene) IsEntityValid(id ecs.ID) bool { return w.IDs.Exists(id) }

// TODO: do we need client-only entities? I don't think we do with this tbh
// TODO: make this private?
func (w *Scene) CreateEntity(info *UpdateParams) ecs.ID {
	if info.Speculating {
		// Create an entity at high index and mark it speculative so that it
		// gets removed when we receive the update for this tick.
		panic("not implemented")
	}
	return w.IDs.Alloc()
}

// This is used by client networking to remove entities.
//
// TODO: is the way we use it correct (deleting entities in-between ticks?)
// TODO: could we bulk delete things?
func (w *Scene) DeleteEntityImmediately(id ecs.ID) {
	columns := reflect.ValueOf(&w.Columns).Elem()
	for i := range columns.NumField() {
		column := columns.Field(i).Addr().Interface().(ecs.AnyColumn).Reflect()
		column.Delete(id)
	}
	if _, ok := w.physicsBodyExists.Get(id); ok {
		w.physicsSystem.RemoveBody(physics.BodyID(id))
		w.physicsBodyExists.Delete(id)
	}
	w.Delete.Delete(id)
	w.IDs.Delete(id)
}

func (w *Scene) Globals() SceneGlobals {
	globals, _ := assertEntity[SceneGlobals](w, 1)
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

func (w *Scene) HandleInput(id ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	if entity, ok := assertEntity[Player](w, id); ok {
		entity.PlayerSubstep(w, id, cmd, info)
	} else {
		info.Logger.Warn(fmt.Sprintf("entity does not exist or does not implement %s", reflect.TypeFor[Player]().Name()), "id", id)
	}
}

// TODO: parallel for in blender for example specifies bulk number for tasks so
// we might want to do the same.

func (w *Scene) Step(updateParams *UpdateParams) {
	w.Now = w.Now.Add(updateParams.Δt)

	// TODO: optimize loops over entities implementing particular interface by
	// having shadow component stores.

	for id, entity := range w.Entity.All() {
		if player, ok := entity.(Player); ok {
			player.PlayerUpdate(w, id, updateParams)
		}
	}

	for id, entity := range w.Entity.All() {
		if entity, ok := entity.(UpdateBeforePhysics); ok {
			entity.UpdateBeforePhysics(w, id, updateParams)
		}
	}

	worldToPhysics(w)
	physicsStep(w, updateParams.Δt)

	for id, entity := range w.Entity.All() {
		if entity, ok := entity.(UpdateAfterPhysics); ok {
			entity.UpdateAfterPhysics(w, id, updateParams)
		}
	}

	// TODO: simulate viewpunch motion better. View punch is a sphere with
	// inertia and damping which we should simulate.
	for id, viewPunch := range w.ViewPunch.All() {
		w.ViewPunch.Set(id, viewPunch.NLerp(geometry.Rot3One(), float32(durationToFloatSeconds(updateParams.Δt))))
	}

	for id, animation := range w.Animation.All() {
		// simple demo, TODO: remove

		action := getAnimation(animation.Action)

		t := float64(w.Now.Sub(animation.PlayTime)) / 1e9

		pose := make(map[string]geometry.Mat4x4)

		// TODO: iterate over a channel map instead
		for channel := range action.samples {
			inverseBindTransform, ok := animation.Armature[channel]
			if !ok {
				continue
			}

			// TODO: we should specify border behavior (e.g. clamp or repeat)
			// TODO: normalized mode (whether entire animation plays in 1 second
			// so the code can specify the time an animation should take) or
			// unnormalized so it plays for as long as was authored.
			pose[channel] = action.Sample(t, channel).Mat4x4().Mul4x4(inverseBindTransform.Mat4x4())
		}

		w.Pose.Set(id, pose)
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

	for id, v := range ecs.Join(w.ContactEvents, w.DeleteCosmeticOffsetOnContact) {
		for _, ce := range v.V1 {
			if ce.Type == 1 {
				w.CosmeticOffset.Delete(id)
				break
			}
		}
	}

	for id, deleteAfter := range w.DeleteAfter.All() {
		if deleteAfter.Before(w.Now) {
			w.Delete.Set(id, struct{}{})
		}
	}

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

		for id := range w.Parent.All() {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	{
		// TODO: should we also clear transientComponents?
		columns := reflect.ValueOf(&w.Columns).Elem()

		for id := range w.Delete.All() {
			for i := range columns.NumField() {
				column := columns.Field(i).Addr().Interface().(ecs.AnyColumn).Reflect()
				column.Delete(id)
			}
			// TODO: delete bodies in bulk
			if _, ok := w.physicsBodyExists.Get(id); ok {
				w.physicsSystem.RemoveBody(physics.BodyID(id))
				w.physicsBodyExists.Delete(id)
			}
			w.Delete.Delete(id)
			w.IDs.Delete(id)
		}
	}
}

// TODO: fold into Update
func ClearTransientComponents(w *Scene) {
	w.ContactEvents.Clear()
}

// TODO: make public
func assertEntity[T any](w *Scene, id ecs.ID) (T, bool) {
	entity, _ := w.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
}
