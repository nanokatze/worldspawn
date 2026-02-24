package game

import (
	"fmt"
	"log"
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/geometry"
	"worldspawn/physics"
)

// TODO: kill these interfaces

type UpdateBeforePhysics interface {
	UpdateBeforePhysics(w *Scene, id ecs.ID, info *UpdateParams)
}

type UpdateAfterPhysics interface {
	UpdateAfterPhysics(w *Scene, id ecs.ID, info *UpdateParams)
}

// TODO: rename to just Layer and move it to worldspawn.go? We'll want another
// component for en/disabling physics if we do so.
// TODO: rename to CollisionFilterGroup or CollisionClass or idk
type CollisionLayer int8

const (
	PhysicsLayerNonMoving CollisionLayer = iota
	PhysicsLayerMoving
	PhysicsLayerProjectiles
	PhysicsLayerMovingKinematic // used by character controllers
	NumPhysicsLayers
)

var collisionLayerMotionType = map[CollisionLayer]int{
	PhysicsLayerNonMoving:       0,
	PhysicsLayerMoving:          2,
	PhysicsLayerProjectiles:     2,
	PhysicsLayerMovingKinematic: 1,
}

var physicsLayerFromString = map[string]CollisionLayer{
	"NonMoving":       PhysicsLayerNonMoving,
	"Moving":          PhysicsLayerMoving,
	"Projectiles":     PhysicsLayerProjectiles,
	"MovingKinematic": PhysicsLayerMovingKinematic,
}

/*
func (physicsLayer *PhysicsLayer) UnmarshalText(text []byte) error {
	tmp, ok := physicsLayerFromString[string(text)]
	if !ok {
		return errors.New("unknown shape type")
	}
	*physicsLayer = tmp
	return nil
}
*/

// TODO: add helpers for filling this out to physics package so that we don't
// need to fill this out explicitly
var ShouldPhysicsLayersCollide = []bool{
	/*                NonM.  Mov.  Proj. */
	/* NonMoving   */ false,
	/* Moving      */ true, true,
	/* Projectiles */ true, true, false,
}

const (
	BroadPhaseLayerNonMoving = iota
	BroadPhaseLayerMoving
	NumBroadPhaseLayers
)

var PhysicsLayerToBroadPhaseLayer = [NumPhysicsLayers]physics.BroadPhaseLayer{
	PhysicsLayerNonMoving:       BroadPhaseLayerNonMoving,
	PhysicsLayerMoving:          BroadPhaseLayerMoving,
	PhysicsLayerProjectiles:     BroadPhaseLayerMoving,
	PhysicsLayerMovingKinematic: BroadPhaseLayerMoving,
}

/*
type ContactKey struct {
	EntityID2   ecs.ID
	SubShapeID1 uint32
	SubShapeID2 uint32
}
*/

type ContactEvent struct {
	Type      int32
	EntityID2 ecs.ID
}

// TODO: add ability to sync per-entity, e.g. we need this to crouch and
// uncrouch the player

// Always execute this system before systems performing physics queries!!!
func (w *Scene) updatePhysicsShadow() {
	for id := range ecs.All(&w.physicsBodyExists) {
		if _, ok := w.CollisionLayer.Get(id); !ok {
			w.physicsSystem.RemoveBody(physics.BodyID(id))
			w.physicsBodyExists.Delete(id)
		}
	}

	for id, layer := range ecs.All(&w.CollisionLayer) {
		translationRotation, _ := w.TranslationRotation.Get(id)
		velocity, _ := w.Velocity.Get(id)
		filter, _ := w.PhysicsFilter.Get(id)

		// TODO: pairwise filter should pass ecs.IDs as is and we should store
		// it on the rigid bodies in the user data slot.
		filter2 := []physics.BodyID{}
		for _, e := range filter {
			_, ok := w.physicsBodyExists.Get(e)
			if !ok {
				continue
			}
			filter2 = append(filter2, physics.BodyID(e))
		}

		motionType2 := collisionLayerMotionType[layer]

		gravityFactor, ok := w.GravityFactor.Get(id)
		if !ok {
			gravityFactor = 1
		}

		shape2 := getShape(w, id)

		mass, overrideMass := w.PhysicsMassOverride.Get(id)
		if !overrideMass {
			mass = shape2.Mass()
		}
		// TODO: non-manifold geometry (which we use for non-moving geo) can get
		// mass=0.
		mass = max(mass, 0.001)

		inertia, overrideInertia := w.PhysicsInertiaOverride.Get(id)
		if !overrideInertia {
			inertia = shape2.Inertia()
		}

		bodyID := physics.BodyID(id)

		_, bodyExists := w.physicsBodyExists.Get(id)
		if !bodyExists {
			w.physicsSystem.AddBody(
				bodyID,
				shape2,
				translationRotation.Translation,
				translationRotation.Rotation,
				velocity.Linear,
				velocity.Angular,
				int(layer),
				filter2,
				motionType2,
				gravityFactor,
				mass,
				inertia)
			w.physicsBodyExists.Set(id, struct{}{})
		} else {
			w.physicsSystem.UpdateBody(
				bodyID,
				shape2,
				translationRotation.Translation,
				translationRotation.Rotation,
				velocity.Linear,
				velocity.Angular,
				int(layer),
				filter2,
				motionType2,
				gravityFactor,
				mass,
				inertia)
		}
	}
}

func getShape(w *Scene, id ecs.ID) *physics.Shape {
	layer, _ := w.CollisionLayer.Get(id)
	geom, _ := w.CollisionGeometry.Get(id)

	motionType2 := collisionLayerMotionType[layer]

	// HACK: our gross way out of not having geonodes, see
	// https://github.com/nanokatze/worldspawn-private/issues/45
	var shape geometryPacked
	switch geom {
	case "Grenade":
		shape = packGeometry(_Geometry{
			Rotation: geometry.Rot3One(),
			Scale:    geometry.Vec3Ones(),

			Kind:       geometrySphere,
			HalfExtent: geometry.Vec3{0.0568, 0.0568, 0.0568},
		})

	case "FPSCharacter":
		shape = packGeometry(_Geometry{
			Translation: geometry.Vec3{0, 0, 1.9 / 2}, // TODO: read standing height off Entity
			Rotation:    geometry.Rot3One(),
			Scale:       geometry.Vec3Ones(),

			Kind:         geometryCylinder,
			HalfExtent:   geometry.Vec3{1, 1, 0}.Scale(0.4).Add(geometry.Vec3{0, 0, 1.9 / 2}),
			ConvexRadius: 0.0,
		})

	default:
		shape = packGeometry(_Geometry{Kind: geometryFileBacked, Filename: geom})
	}

	var shape2 *physics.Shape
	if motionType2 == 0 {
		shape2 = getConcaveShape(shape)
	} else {
		shape2 = getConvexShape(shape)
	}

	return shape2
}

// TODO: we could split this back so that we can run stuff in parallel
func (w *Scene) physicsStep(Δt time.Duration) {
	w.physicsSystem.SetGravity(w.Globals().Gravity)
	w.physicsSystem.Update(float32(durationToFloatSeconds(Δt)))

	for _, bodyID := range w.physicsSystem.ActiveBodies() {
		entityID := ecs.ID(bodyID) // BUG: this is not correct anymore because the generations do not match!!!

		pos, rot, linVel, angVel := w.physicsSystem.WritebackBody(bodyID)

		w.TranslationRotation.Set(entityID, TranslationRotation{Translation: pos, Rotation: rot})

		// TODO: don't store velocity back for kinematic bodies
		w.Velocity.Set(entityID, Velocity{Linear: linVel, Angular: angVel})
	}

	for _, ce := range w.physicsSystem.ContactEvents() {
		entityID1 := ecs.ID(ce.Body1.BodyID)
		entityID2 := ecs.ID(ce.Body2.BodyID)

		// One or both entities in the contact event might've been removed
		// before the last Update

		if w.IsEntityValid(entityID1) {
			contactEvents, _ := w.ContactEvents.Get(entityID1)
			contactEvents = append(contactEvents, ContactEvent{
				Type:      ce.Type,
				EntityID2: entityID2,
			})
			w.ContactEvents.Set(entityID1, contactEvents)
		}

		if w.IsEntityValid(entityID2) {
			contactEvents, _ := w.ContactEvents.Get(entityID2)
			contactEvents = append(contactEvents, ContactEvent{
				Type:      ce.Type,
				EntityID2: entityID1,
			})
			w.ContactEvents.Set(entityID2, contactEvents)
		}
	}
}

// TODO: don't duplicate things we don't need to.

var convexCache = make(map[geometryPacked]*physics.Shape)
var concaveCache = make(map[geometryPacked]*physics.Shape)

func getConvexShape(key2 geometryPacked) *physics.Shape {
	shape, ok := convexCache[key2]
	if ok {
		return shape
	}

	key := unpackGeometry(key2)

	var err error
	switch key.Kind {
	case geometrySphere:
		shape, err = physics.NewSphereShape(key.HalfExtent[0])

	case geometryBox:
		shape, err = physics.NewBoxShape(key.HalfExtent, key.ConvexRadius)

	case geometryCylinder:
		shape, err = physics.NewCylinderShape(key.HalfExtent[0], key.HalfExtent[2], key.ConvexRadius)

	case geometryFileBacked:
		shape, err = physics.NewFileBackedShape(Data, key.Filename, false)

	default:
		panic(fmt.Sprintf("unknown physics shape kind %v", key.Kind))
	}
	if err != nil {
		// TODO: actually print a warning and return a box?
		log.Fatal(err)
	}
	shape, err = physics.NewTransformedShape(key.Translation, key.Rotation, key.Scale, shape)
	if err != nil {
		// TODO: actually print a warning and return a box
		log.Fatal(err)
	}
	convexCache[key2] = shape
	return shape
}

func getConcaveShape(key2 geometryPacked) *physics.Shape {
	shape, ok := concaveCache[key2]
	if ok {
		return shape
	}

	key := unpackGeometry(key2)

	var err error
	switch key.Kind {
	case geometrySphere:
		shape, err = physics.NewSphereShape(key.HalfExtent[0])

	case geometryBox:
		shape, err = physics.NewBoxShape(key.HalfExtent, key.ConvexRadius)

	case geometryCylinder:
		shape, err = physics.NewCylinderShape(key.HalfExtent[0], key.HalfExtent[2], key.ConvexRadius)

	case geometryFileBacked:
		shape, err = physics.NewFileBackedShape(Data, key.Filename, true)

	default:
		panic(fmt.Sprintf("unknown physics shape kind %v", key.Kind))
	}
	if err != nil {
		// TODO: actually print a warning and return a box?
		log.Fatal(err)
	}
	shape, err = physics.NewTransformedShape(key.Translation, key.Rotation, key.Scale, shape)
	if err != nil {
		// TODO: actually print a warning and return a box
		log.Fatal(err)
	}
	concaveCache[key2] = shape
	return shape
}
