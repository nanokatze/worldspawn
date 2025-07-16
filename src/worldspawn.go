package worldspawn

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/go-json-experiment/json"

	"worldspawn/ecs"
	"worldspawn/ecs/bitset"
	"worldspawn/experiments/encoding/nice"
	"worldspawn/geometry-go"
	"worldspawn/physics"
)

// TODO: move these things to their own files

// TODO: it is up to the server and client to implement un/marshalers, we should
// only expose the info necessary for it
var WorldJSONOptions = json.JoinOptions(
	JSONOptions,

	json.WithMarshalers(json.JoinMarshalers(
		json.MarshalToFunc(entityJSONMarshaler),
	)),

	json.WithUnmarshalers(json.JoinUnmarshalers(
		json.UnmarshalFromFunc(entityJSONUnmarshaler),
	)),
)

// TODO: same as WorldJSONOptions
var WorldNiceOptions = nice.JoinOptions(
	nice.WithMarshaler(EntityNiceMarshaler),
	nice.WithUnmarshaler(EntityNiceUnmarshaler),
)

type UpdateFlags int

const (
	UpdateSpeculative UpdateFlags = 1 << iota
)

var Data = os.DirFS("data/cooked") // TODO: this should be set per-World (which has implications for caching, etc)

// TODO: something to let us control what gets sent to a client.

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

type World struct {
	SingletonComponents

	IDAlloc *ecs.IDAlloc
	Components
	TransientComponents

	physicsSystem      *physics.System
	physicsBodyExists2 bitset.Bitset // TODO: rename
	physicsBodyExists  ecs.ComponentStore[struct{}]
}

type SingletonComponents struct {
	Now Time

	// TODO: replace it with full blown sky material struct/map?
	//
	// TODO: make this a non-singleton component so we can have different parts
	// of the map have different skies? Yes, actually, we'll probably want to
	// make a "Scene" entity that envelopes a scene. Scenes will live in their
	// separate TLASes so Sky can always be just a miss shader.
	Sky string

	Gravity geometry.Vec3
}

// TODO: introduce Camera component which will specify fov etc

// TODO: component for viewmodels and being drawn at custom fov or something
// TODO: we'll probably want viewmodels to be separate entities

// TODO: add a type for representing non-mutable lists and stuff?

// TODO: make Components be registered dynamically? this would let us move them
// into each individual func init()

// TODO: we can update OldWorld (against which we compute delta) only sometimes
// and incrementally (this is the actually useful bit, as updating "sometimes"
// will make certain ticks longer, potentially making them miss deadline.) we
// should also update it reactively when we add dirty tracking, to minimize the
// amount of work we do.

// TODO: introduce struct tags like compatibility names etc
type Components struct {
	Entity ecs.ComponentStore[Entity]

	Parent   ecs.ComponentStore[ecs.ID]
	Children ecs.ComponentStore[[]ecs.ID] // or map[ecs.ID]struct{} ?

	TranslationRotation ecs.ComponentStore[TranslationRotation]
	Scale               ecs.ComponentStore[geometry.Vec3] // TODO: make it so that if scale is not provided, we default to 1? Yes, and rename it to ScaleOverride or CustomScale

	Velocity ecs.ComponentStore[Velocity]

	// TODO: rename to RenderingModel
	//
	// TODO: renderer model and physics shape models should share the container
	// (i.e. it should be the same .mdl file)
	RendererModel ecs.ComponentStore[RendererModel]

	SoundEffect ecs.ComponentStore[SoundEffect]

	CosmeticOffset ecs.ComponentStore[CosmeticOffset]

	// Posing test
	Animation ecs.ComponentStore[Animation]
	Pose      ecs.ComponentStore[map[string]geometry.Mat4x4]

	// NOTE: constraints and pairwise filter
	//
	// Should we have an identifier for each filtered/constrained entity so that
	// we can have multiple filters against the same entity? Another option
	// would be to have any entity specify filtered/constrained pairs. Yet
	// another would be to have constraints and nocollide pairs be its own
	// concept in the world
	//
	// TODO: merge some of these components?

	// TODO: rename to ColliderModel or CollisionModel? Or something
	PhysicsShape           ecs.ComponentStore[PhysicsShape]
	PhysicsLayer           ecs.ComponentStore[PhysicsLayer]
	PhysicsMotionType      ecs.ComponentStore[PhysicsMotionType]
	PhysicsFilter          ecs.ComponentStore[[]ecs.ID] // TODO: rename to something like PairwiseFilters?
	GravityFactor          ecs.ComponentStore[float32]
	PhysicsMassOverride    ecs.ComponentStore[float32] // TODO: remove "Physics" prefix from these
	PhysicsInertiaOverride ecs.ComponentStore[geometry.Mat4x4]

	ArmedCharacter ecs.ComponentStore[ArmedCharacter]
	// TODO: unify these two components probably
	ViewPunch         ecs.ComponentStore[geometry.Rot3]
	ViewPunchVelocity ecs.ComponentStore[geometry.Vec3]

	Viewmodel ecs.ComponentStore[Viewmodel]

	PlayerSpawn                  ecs.ComponentStore[struct{}]
	ResetCosmeticOffsetOnContact ecs.ComponentStore[struct{}]
	SpawnTime                    ecs.ComponentStore[Time]
	DeleteAfter                  ecs.ComponentStore[Time]

	// Prefab ecs.ComponentStore[*Components]

	// Timer ecs.ComponentStore[time.Duration]
}

type TransientComponents struct {
	// TODO: get rid of this component
	WeaponAim ecs.ComponentStore[WeaponAim]

	// TODO: generalize to all events, including damage etc?
	ContactEvents ecs.ComponentStore[[]ContactEvent]

	Delete ecs.ComponentStore[struct{}]
}

func NewWorld(n int) *World {
	w := new(World)

	w.IDAlloc = ecs.NewIDAlloc(n)

	// TODO: make it clear that these are reflect references

	components := reflect.ValueOf(&w.Components).Elem()
	for i := range components.Type().NumField() {
		components.Field(i).Addr().Interface().(interface{ Init(idAlloc *ecs.IDAlloc) }).Init(w.IDAlloc)
	}

	transientComponents := reflect.ValueOf(&w.TransientComponents).Elem()
	for i := range transientComponents.Type().NumField() {
		transientComponents.Field(i).Addr().Interface().(interface{ Init(idAlloc *ecs.IDAlloc) }).Init(w.IDAlloc)
	}

	w.physicsSystem = physics.NewSystem(
		int(NumBroadPhaseLayers),
		int(NumPhysicsLayers),
		PhysicsLayerToBroadPhaseLayer,
		ShouldPhysicsLayersCollide)
	// This will have to be a raw bitmap or just have its own IDAlloc
	w.physicsBodyExists2 = bitset.Make(n)
	w.physicsBodyExists.Init(w.IDAlloc)

	// TODO: we should expose an OptimizeBroadPhase call on physicsSystem which
	// we'll (optionally) call after loading the world and perhaps every so
	// often

	return w
}

func (w *World) Destroy() {
	// TODO: stop and destroy physicsSystem here
}

func (w *World) IsEntityValid(id ecs.ID) bool {
	return w.IDAlloc.Valid(id)
}

func (w *World) SpawnEntity() ecs.ID {
	return w.IDAlloc.Alloc()
}

// This is used by client networking to remove entities.
//
// TODO: is the way we use it correct (deleting entities in-between ticks?)
// TODO: could we bulk delete things?
func (w *World) DeleteEntityImmediately(id ecs.ID) {
	components := reflect.ValueOf(w).Elem().FieldByName("Components")
	for i := range components.NumField() {
		components.Field(i).Addr().Interface().(interface{ Delete(ecs.ID) }).Delete(id)
	}
	if _, ok := w.physicsBodyExists.Load(id); ok {
		w.physicsSystem.RemoveBody(physics.BodyID(id))
		w.physicsBodyExists.Delete(id)
	}
	w.Delete.Delete(id)
	w.IDAlloc.Free(id)
}

// TODO: rename to User/Player/etc Input
// TODO: does Δt make any sense here?
func (w *World) HandleInput(id ecs.ID, cmd TimestampedInputCmd, Δt time.Duration, flags UpdateFlags, logger *slog.Logger) {
	if entity, ok := loadEntity[Character](w, id); ok {
		entity.CharacterUpdate(w, id, cmd, Δt, flags)
	} else {
		logger.Warn(fmt.Sprintf("entity does not exist or does not implement %s", reflect.TypeFor[Character]().Name()), "id", id)
	}
}

// TODO: parallel for in blender for example specifies bulk number for tasks so
// we might want to do the same.

func (w *World) Update(Δt time.Duration, flags UpdateFlags, logger *slog.Logger) {
	if w.Now == 0 {
		panic("Now must never be zero")
	}

	// TODO: optimize loops over entities implementing particular interface by
	// having shadow component stores.

	for id, entity := range w.Entity.All() {
		if char, ok := entity.(Character); ok {
			// TODO: not sure what to put in Time here, w.Now or w.Now + Δt?
			char.CharacterUpdate(w, id, TimestampedInputCmd{Time: 0}, Δt, flags)
		}
	}

	for id, entity := range w.Entity.All() {
		if entity, ok := entity.(UpdateBeforePhysics); ok {
			entity.UpdateBeforePhysics(w, id, Δt)
		}
	}

	worldToPhysics(w)
	updatePhysics(w, Δt)

	for id, entity := range w.Entity.All() {
		if entity, ok := entity.(UpdateAfterPhysics); ok {
			entity.UpdateAfterPhysics(w, id, Δt, flags)
		}
	}

	// TODO: simulate viewpunch motion better. View punch is a sphere with
	// inertia and damping which we should simulate.
	for id, viewPunch := range w.ViewPunch.All() {
		w.ViewPunch.Store(id, viewPunch.NLerp(geometry.Rot3One(), float32(durationToFloatSeconds(Δt))))
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

		w.Pose.Store(id, pose)
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

	for id, v := range ecs.Join(w.ContactEvents, w.ResetCosmeticOffsetOnContact) {
		for _, ce := range v.V1 {
			if ce.Type == 1 {
				w.CosmeticOffset.Delete(id)
				break
			}
		}
	}

	for id, deleteAfter := range w.DeleteAfter.All() {
		if deleteAfter.Before(w.Now) {
			w.Delete.Store(id, struct{}{})
		}
	}

	// TODO: would we benefit from (optionally) checking whether any entities
	// have dangling references to other entities?

	// TODO: delete entities too far off the map

	// Remove entities that were scheduled for removal
	{
		// TODO: should we also clear transientComponents?
		components := reflect.ValueOf(w).Elem().FieldByName("Components")
		fields := components.Type().NumField()

		for id := range w.Delete.All() {
			for i := 0; i < fields; i++ {
				components.Field(i).Addr().Interface().(interface{ Delete(ecs.ID) }).Delete(id)
			}
			// TODO: delete bodies in bulk
			if _, ok := w.physicsBodyExists.Load(id); ok {
				w.physicsSystem.RemoveBody(physics.BodyID(id))
				w.physicsBodyExists.Delete(id)
			}
			w.Delete.Delete(id)
			w.IDAlloc.Free(id)
		}
	}
}

func ClearTransientComponents(w *World) {
	// TODO: uncomment this when we find a good way to draw view models
	// w.WeaponAim.Clear()

	w.ContactEvents.Clear()
}

// TODO: rename this helper to expectEntity or something
func loadEntity[T any](w *World, id ecs.ID) (T, bool) {
	entity, ok := w.Entity.Load(id)
	if !ok {
		return *new(T), false
	}

	entityT, ok := entity.(T)
	if !ok {
		return *new(T), false
	}

	return entityT, true
}
