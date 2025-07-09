package worldspawn

import (
	"time"

	"worldspawn/ecs"
	"worldspawn/physics"
)

type UpdateBeforePhysics interface {
	// TODO: should also have updateFlags
	UpdateBeforePhysics(w *World, id ecs.ID, Δt time.Duration)
}

type UpdateAfterPhysics interface {
	UpdateAfterPhysics(w *World, id ecs.ID, Δt time.Duration, updateFlags UpdateFlags)
}

// TODO: change these to be UnmarshalText and MarshalText so that we don't have
// to author with numbers.

type PhysicsMotionType int8

const (
	PhysicsMotionStatic PhysicsMotionType = iota
	PhysicsMotionDynamic
	PhysicsMotionKinematic
	// TODO: another Kinematic movetype where we only set the origin?
)

type PhysicsLayer int8

const (
	PhysicsLayerNonMoving = iota
	PhysicsLayerMoving
	PhysicsLayerProjectiles
	NumPhysicsLayers
)

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

var PhysicsLayerToBroadPhaseLayer = []physics.BroadPhaseLayer{
	PhysicsLayerNonMoving:   BroadPhaseLayerNonMoving,
	PhysicsLayerMoving:      BroadPhaseLayerMoving,
	PhysicsLayerProjectiles: BroadPhaseLayerMoving,
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
	w.physicsBodyExists.All()(func(entityID ecs.ID, _ struct{}) bool {
		if _, ok := w.PhysicsShape.Load(entityID); !ok {
			w.physicsSystem.RemoveBody(physics.BodyID(entityID))
			w.physicsBodyExists.Delete(entityID)
		}
		return true
	})

	for entityID, shape := range w.PhysicsShape.All() {
		translationRotation, _ := w.TranslationRotation.Load(entityID)
		velocity, _ := w.Velocity.Load(entityID)
		layer, _ := w.PhysicsLayer.Load(entityID)
		motionType, _ := w.PhysicsMotionType.Load(entityID)
		filter, _ := w.PhysicsFilter.Load(entityID)

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

		motionType2 := []int{0, 2, 1}[motionType]

		gravityFactor, ok := w.GravityFactor.Load(entityID)
		if !ok {
			gravityFactor = 1
		}

		shape2 := getShape(shape)

		mass, overrideMass := w.PhysicsMassOverride.Load(entityID)
		if !overrideMass {
			mass = shape2.Mass()
		}

		inertia, overrideInertia := w.PhysicsInertiaOverride.Load(entityID)
		if !overrideInertia {
			inertia = shape2.Inertia()
		}

		bodyID := physics.BodyID(entityID)

		_, bodyExists := w.physicsBodyExists.Load(entityID)
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
			w.physicsBodyExists.Store(entityID, struct{}{})
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
