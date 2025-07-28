package worldspawn

import (
	"time"

	"worldspawn/ecs"
	"worldspawn/physics"
)

// TODO: rename to BeforePhysics
type UpdateBeforePhysics interface {
	UpdateBeforePhysics(w *World, id ecs.ID, info *UpdateParams)
}

// TODO: rename to AfterPhysics
type UpdateAfterPhysics interface {
	UpdateAfterPhysics(w *World, id ecs.ID, info *UpdateParams)
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
// TODO: a more descriptive name
func worldToPhysics(w *World) {
	for id := range w.physicsBodyExists.All() {
		if _, ok := w.CollisionLayer.Load(id); !ok {
			w.physicsSystem.RemoveBody(physics.BodyID(id))
			w.physicsBodyExists.Delete(id)
		}
	}

	for id, layer := range w.CollisionLayer.All() {
		translationRotation, _ := w.TranslationRotation.Load(id)
		velocity, _ := w.Velocity.Load(id)
		shape, _ := w.CollisionGeometry.Load(id)
		filter, _ := w.PhysicsFilter.Load(id)

		// TODO: pairwise filter should pass ecs.IDs as is and we should store
		// it on the rigid bodies in the user data slot.
		filter2 := []physics.BodyID{}
		for _, e := range filter {
			_, ok := w.physicsBodyExists.Load(e)
			if !ok {
				continue
			}
			filter2 = append(filter2, physics.BodyID(e))
		}

		motionType2 := collisionLayerMotionType[layer]

		gravityFactor, ok := w.GravityFactor.Load(id)
		if !ok {
			gravityFactor = 1
		}

		shape2 := getShape(shape)

		mass, overrideMass := w.PhysicsMassOverride.Load(id)
		if !overrideMass {
			mass = shape2.Mass()
		}
		// TODO: non-manifold geometry (which we use for non-moving geo) can get
		// mass=0.
		mass = max(mass, 0.001)

		inertia, overrideInertia := w.PhysicsInertiaOverride.Load(id)
		if !overrideInertia {
			inertia = shape2.Inertia()
		}

		bodyID := physics.BodyID(id)

		_, bodyExists := w.physicsBodyExists.Load(id)
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
			w.physicsBodyExists.Store(id, struct{}{})
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

// TODO: we could split this back so that we can run stuff in parallel
func updatePhysics(w *World, Δt time.Duration) {
	w.physicsSystem.SetGravity(w.Gravity)
	w.physicsSystem.Update(float32(durationToFloatSeconds(Δt)))

	for _, bodyID := range w.physicsSystem.ActiveBodies() {
		entityID := ecs.ID(bodyID) // BUG: this is not correct anymore because the generations do not match!!!

		pos, rot, linVel, angVel := w.physicsSystem.WritebackBody(bodyID)

		w.TranslationRotation.Store(entityID, TranslationRotation{Translation: pos, Rotation: rot})

		// TODO: don't store velocity back for kinematic bodies
		w.Velocity.Store(entityID, Velocity{Linear: linVel, Angular: angVel})
	}

	for _, ce := range w.physicsSystem.ContactEvents() {
		entityID1 := ecs.ID(ce.Body1.BodyID)
		entityID2 := ecs.ID(ce.Body2.BodyID)

		// One or both entities in the contact event might've been removed
		// before the last Update

		if w.IsEntityValid(entityID1) {
			contactEvents, _ := w.ContactEvents.Load(entityID1)
			contactEvents = append(contactEvents, ContactEvent{
				Type:      ce.Type,
				EntityID2: entityID2,
			})
			w.ContactEvents.Store(entityID1, contactEvents)
		}

		if w.IsEntityValid(entityID2) {
			contactEvents, _ := w.ContactEvents.Load(entityID2)
			contactEvents = append(contactEvents, ContactEvent{
				Type:      ce.Type,
				EntityID2: entityID1,
			})
			w.ContactEvents.Store(entityID2, contactEvents)
		}
	}
}
