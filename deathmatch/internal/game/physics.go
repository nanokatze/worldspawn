package game

import (
	"fmt"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

type Velocity struct {
	Linear  gmath.Vec3f32
	Angular gmath.Vec3f32 // TODO: this should be a scalar per basis plane rather than vec3
}

func (a Velocity) Add(b Velocity) Velocity {
	return Velocity{
		Linear:  a.Linear.Add(b.Linear),
		Angular: a.Angular.Add(b.Angular),
	}
}

/*
type ContactKey struct {
	EntityID2   ecs.ID
	SubShapeID1 uint32
	SubShapeID2 uint32
}
*/

// TODO: have different types depending on contact added/removed/etc
type ContactEvent struct {
	Type      int32
	EntityID2 ecs.ID
}

func (contact ContactEvent) Apply(world *World, id ecs.ID, updateParams *UpdateParams) {
	if contact.Type == 1 {
		if _, ok := world.DeleteCosmeticOffsetOnContact.Get(id); ok {
			world.CosmeticOffset.Delete(id)
			world.DeleteCosmeticOffsetOnContact.Delete(id)
		}
	}

	// TODO: pass it to the script
}

// TODO: add ability to sync per-entity, e.g. we need this to crouch and
// uncrouch the player I think

// Always run this before performing physics queries!!!
func (world *World) updatePhysicsShadow(updateParams *UpdateParams) {
	// TODO: remove bodies when we delete entities!!!!!!!!
	for id := range ecs.All(&world.physicsBodyExists) {
		if _, ok := world.CollisionLayer.Get(id); !ok {
			world.physics.RemoveBody(physics.BodyID(id))
			world.physicsBodyExists.Delete(id)
		}
	}

	for id, layer := range ecs.All(&world.CollisionLayer) {
		trs := world.GetTransform(id)
		// TODO: ensure trs.S is 1
		velocity, _ := world.Velocity.Get(id)
		filter, _ := world.PhysicsFilter.Get(id)

		// TODO: pairwise filter should pass ecs.IDs as is and we should store
		// it on the rigid bodies in the user data slot.
		filter2 := []physics.BodyID{}
		for _, e := range filter {
			_, ok := world.physicsBodyExists.Get(e)
			if !ok {
				continue
			}
			filter2 = append(filter2, physics.BodyID(e))
		}

		motionType2 := collisionLayerMotionType[layer]

		gravityFactor, ok := world.GravityFactor.Get(id)
		if !ok {
			gravityFactor = 1
		}

		shape2 := getShape(world, id)

		mass, overrideMass := world.PhysicsMassOverride.Get(id)
		if !overrideMass {
			mass = shape2.Mass()
		}
		// TODO: non-manifold geometry (which we use for non-moving geo) can get
		// mass=0.
		mass = max(mass, 0.001)

		inertia, overrideInertia := world.PhysicsInertiaOverride.Get(id)
		if !overrideInertia {
			inertia = shape2.Inertia()
		}

		bodyID := physics.BodyID(id.Index())

		_, bodyExists := world.physicsBodyExists.Get(id)
		if !bodyExists {
			world.physics.AddBody(
				bodyID,
				shape2,
				trs.T,
				trs.R,
				velocity.Linear,
				velocity.Angular,
				int(layer),
				filter2,
				motionType2,
				gravityFactor,
				mass,
				inertia)
			world.physicsBodyExists.Set(id, struct{}{})
		} else {
			world.physics.UpdateBody(
				bodyID,
				shape2,
				trs.T,
				trs.R,
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

func getShape(world *World, id ecs.ID) *physics.Shape {
	layer, _ := world.CollisionLayer.Get(id)
	geom, _ := world.CollisionGeometry.Get(id)

	motionType2 := collisionLayerMotionType[layer]

	// HACK: our gross way out of not having geonodes, see
	// https://github.com/nanokatze/worldspawn-private/issues/45
	var shape geometryPacked
	switch geom {
	case "Grenade":
		shape = packGeometry(_Geometry{
			Rotation: gmath.Rot3One(),
			Scale:    gmath.Vec3Ones[float32](),

			Kind:       geometrySphere,
			HalfExtent: gmath.Vec3f32{0.0568, 0.0568, 0.0568},
		})

	case "FPSCharacter":
		shape = packGeometry(_Geometry{
			Translation: gmath.Vec3f32{0, 0, 1.9 / 2}, // TODO: read standing height off Entity
			Rotation:    gmath.Rot3One(),
			Scale:       gmath.Vec3Ones[float32](),

			Kind:         geometryCylinder,
			HalfExtent:   gmath.Vec3f32{1, 1, 0}.Scale(0.4).Add(gmath.Vec3f32{0, 0, 1.9 / 2}),
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

func (world *World) physicsStep(updateParams *UpdateParams) {
	// TODO: push it back onto the user again?
	world.updatePhysicsShadow(updateParams)

	world.physics.SetGravity(world.Globals().Gravity)
	world.physics.Update(float32(durationToFloatSeconds(updateParams.Δt)))

	for _, bodyID := range world.physics.ActiveBodies() {
		entityID := ecs.ID(bodyID) // BUG: this is not correct anymore because the generations do not match!!!

		pos, rot, linVel, angVel := world.physics.WritebackBody(bodyID)

		if !world.EntityExists(entityID) {
			updateParams.Logger.Info("entity does not exist for some reason", "id", entityID)
		}

		world.TransformTR.Set(entityID, TR3f64{T: pos, R: rot})

		// TODO: don't store velocity back for kinematic bodies
		world.Velocity.Set(entityID, Velocity{Linear: linVel, Angular: angVel})
	}

	for _, ce := range world.physics.ContactEvents() {
		// TODO: properly translate bodyID to entityID
		entityID1 := ecs.ID(ce.Body1.BodyID)
		entityID2 := ecs.ID(ce.Body2.BodyID)

		// TODO: spawn sounds on collision

		// One or both entities in the contact event might've been removed
		// before the last Update

		if world.EntityExists(entityID1) && world.EntityExists(entityID2) {
			world.EnqueueEntityUpdate(entityID1, ContactEvent{Type: ce.Type, EntityID2: entityID2}.Apply)
			world.EnqueueEntityUpdate(entityID2, ContactEvent{Type: ce.Type, EntityID2: entityID1}.Apply)
		}
	}

	world.processEntityUpdates(updateParams)
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
		panic(err)
	}
	shape, err = physics.NewTransformedShape(key.Translation, key.Rotation, key.Scale, shape)
	if err != nil {
		// TODO: actually print a warning and return a box
		panic(err)
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
		panic(err)
	}
	shape, err = physics.NewTransformedShape(key.Translation, key.Rotation, key.Scale, shape)
	if err != nil {
		// TODO: actually print a warning and return a box
		panic(err)
	}
	concaveCache[key2] = shape
	return shape
}
