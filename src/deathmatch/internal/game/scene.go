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

type UpdateParams struct {
	Δt          time.Duration
	Speculating bool
	Logger      *slog.Logger
}

/*
type SubtickUpdateParams struct {
	UpdateParams
}
*/

// TODO: move this into the World object?
var Data fs.FS

// TODO: something to let us control what gets sent to a client.

// TODO: rename to TranslationAndRotation?
type TranslationRotation struct {
	Translation geometry.DVec3
	Rotation    geometry.Rot3
}

func TranslationRotationOne() TranslationRotation {
	return TranslationRotation{
		Translation: geometry.DVec3{},
		Rotation:    geometry.Rot3One(),
	}
}

type Velocity struct {
	Linear  geometry.Vec3
	Angular geometry.Vec3
}

/*
func (w *World) Transform(id ecs.ID) (geometry.Mat4x4, bool) {
	A, ok := w.TranslationRotation.Load(id)
	return A, ok
}
*/

// TODO: rename?
// TODO: it's annoying that this needs special handling. We can replace most of
// these with just reading things from entity 1. The time would have to remain
// special though.
type SingletonComponents struct {
	// TODO: replace it with sky material
	Sky string

	Gravity geometry.Vec3
}

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

	CreationTime ecs.Column[Time]

	Parent ecs.Column[ecs.Entity]
	// Children ecs.ComponentStore[map[ecs.ID] // map[ecs.ID]struct{} ?

	TranslationRotation ecs.Column[TranslationRotation]
	Scale               ecs.Column[geometry.Vec3] // TODO: default this to 1

	Velocity ecs.Column[Velocity]

	RenderingGeometry ecs.Column[GeometryPacked]

	// TODO: rename to SoundEmitter
	SoundEffect ecs.Column[SoundEmitter]

	Viewmodel2 ecs.Column[Viewmodel2]

	CosmeticOffset                ecs.Column[CosmeticOffset]
	DeleteCosmeticOffsetOnContact ecs.Column[struct{}]

	// Posing test
	Animation ecs.Column[Animation]
	Pose      ecs.Column[map[string]geometry.Mat4x4]

	// NOTE: constraints and pairwise filter
	//
	// Should we have an identifier for each filtered/constrained entity so that
	// we can have multiple filters against the same entity? Another option
	// would be to have any entity specify filtered/constrained pairs. Yet
	// another would be to have constraints and nocollide pairs be its own
	// concept in the world
	//

	// TODO: merge some of these components?

	CollisionLayer         ecs.Column[CollisionLayer]
	CollisionGeometry      ecs.Column[GeometryPacked]
	PhysicsFilter          ecs.Column[[]ecs.Entity] // TODO: rename to something like PairwiseFilters?
	GravityFactor          ecs.Column[float32]
	PhysicsMassOverride    ecs.Column[float32] // TODO: remove "Physics" prefix from these
	PhysicsInertiaOverride ecs.Column[geometry.Mat4x4]

	// TODO: remove this component
	ArmedCharacter ecs.Column[ArmedCharacter]

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

// GlobalProperties?
type SceneProperties struct {
	Sky     string
	Gravity geometry.Vec3
}

type Scene struct {
	// TODO: document what this means when we're in the middle of an Update
	Now Time
	SingletonComponents

	Entities *ecs.Entities
	Columns
	physicsSystem     *physics.System
	physicsBodyExists ecs.Column[struct{}]
}

func NewScene(n int) *Scene {
	w := new(Scene)

	w.Entities = ecs.NewEntities(n)

	// TODO: make it clear that these are reflect references

	columns := reflect.ValueOf(&w.Columns).Elem()
	for i := range columns.Type().NumField() {
		columns.Field(i).Addr().Interface().(interface{ Init(*ecs.Entities) }).Init(w.Entities)
	}

	w.physicsSystem = physics.NewSystem(
		int(NumBroadPhaseLayers),
		int(NumPhysicsLayers),
		PhysicsLayerToBroadPhaseLayer[:],
		ShouldPhysicsLayersCollide)
	w.physicsBodyExists.Init(w.Entities)

	// TODO: we should expose an OptimizeBroadPhase call on physicsSystem which
	// we'll (optionally) call after loading the world and perhaps every so
	// often

	return w
}

func (w *Scene) Destroy() {
	// TODO: stop and destroy physicsSystem here
}

// TODO: rename to EntityIsValid?
func (w *Scene) IsEntityValid(id ecs.Entity) bool {
	return w.Entities.Valid(id)
}

// TODO: do we need client-only entities? I don't think we do with this tbh
// TODO: make this private?
func (w *Scene) CreateEntity(info *UpdateParams) ecs.Entity {
	if info.Speculating {
		// Create an entity at high index and mark it speculative so that it
		// gets removed when we receive the update for this tick.
		panic("not implemented")
	}
	return w.Entities.Alloc()
}

// This is used by client networking to remove entities.
//
// TODO: is the way we use it correct (deleting entities in-between ticks?)
// TODO: could we bulk delete things?
func (w *Scene) DeleteEntityImmediately(id ecs.Entity) {
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
	w.Entities.Free(id)
}

// TODO: or SetParent?
func (scene *Scene) ParentTo(child, parent ecs.Entity) {
	scene.Parent.Set(child, parent)

	// children, _ := scene.Children.Load(parent)
	// children
}

func (scene *Scene) GetScale(id ecs.Entity) geometry.Vec3 {
	if scale, ok := scene.Scale.Get(id); ok {
		return scale
	}
	return geometry.Vec3Broadcast(1)
}

// TODO: rename to User/Player/etc Input
func (w *Scene) HandleInput(id ecs.Entity, cmd TimestampedInputCmd, info *UpdateParams) {
	if entity, ok := assertEntity[Character](w, id); ok {
		entity.CharacterUpdate(w, id, cmd, info)
	} else {
		info.Logger.Warn(fmt.Sprintf("entity does not exist or does not implement %s", reflect.TypeFor[Character]().Name()), "id", id)
	}
}

// TODO: parallel for in blender for example specifies bulk number for tasks so
// we might want to do the same.

func (w *Scene) Update(updateParams *UpdateParams) {
	w.Now = w.Now.Add(updateParams.Δt)

	// TODO: optimize loops over entities implementing particular interface by
	// having shadow component stores.

	for id, entity := range w.Entity.All() {
		if char, ok := entity.(Character); ok {
			// TODO: not sure what to put in Time here, w.Now or w.Now + Δt?
			char.CharacterUpdate(w, id, TimestampedInputCmd{Time: 0}, updateParams)
		}
	}

	for id, entity := range w.Entity.All() {
		if entity, ok := entity.(UpdateBeforePhysics); ok {
			entity.UpdateBeforePhysics(w, id, updateParams)
		}
	}

	worldToPhysics(w)
	updatePhysics(w, updateParams.Δt)

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

	// Remove entities that were scheduled for removal
	{
		// TODO: should we also clear transientComponents?
		columns := reflect.ValueOf(&w.Columns).Elem()
		fields := columns.Type().NumField()

		for id := range w.Delete.All() {
			for i := range fields {
				column := columns.Field(i).Addr().Interface().(ecs.AnyColumn).Reflect()
				column.Delete(id)
			}
			// TODO: delete bodies in bulk
			if _, ok := w.physicsBodyExists.Get(id); ok {
				w.physicsSystem.RemoveBody(physics.BodyID(id))
				w.physicsBodyExists.Delete(id)
			}
			w.Delete.Delete(id)
			w.Entities.Free(id)
		}
	}
}

// TODO: fold into Update
func ClearTransientComponents(w *Scene) {
	// TODO: uncomment this when we find a good way to draw view models
	w.ContactEvents.Clear()
}

// TODO: make public
func assertEntity[T any](w *Scene, id ecs.Entity) (T, bool) {
	entity, _ := w.Entity.Get(id)
	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}
	return entityT, true
}
