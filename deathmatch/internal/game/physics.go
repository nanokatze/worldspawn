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

func (a Velocity) Scale(λ float32) Velocity {
	return Velocity{
		Linear:  a.Linear.Scale(λ),
		Angular: a.Angular.Scale(λ),
	}
}

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
		_, sensor := world.CollisionSensor.Get(id)

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
				motionType2,
				gravityFactor,
				mass,
				inertia,
				sensor)
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
				motionType2,
				gravityFactor,
				mass,
				inertia,
				sensor)
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

	case "Gladiator":
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

	world.physics.Update(float32(durationToFloatSeconds(updateParams.Δt)), world.Globals().Gravity)

	for _, bodyID := range world.physics.ActiveBodies() {
		entityID := ecs.ID(bodyID) // BUG: this is not correct anymore because the generations do not match!!!

		pos, rot, linVel, angVel := world.physics.WritebackBody(bodyID)

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
			switch ce.Type {
			case 1:
				if script := world.GetScriptFuncs(entityID1); script.ContactAdded != nil {
					script.ContactAdded(world, entityID1, entityID2, updateParams)
				}
				if script := world.GetScriptFuncs(entityID2); script.ContactAdded != nil {
					script.ContactAdded(world, entityID2, entityID1, updateParams)
				}

			case 2:
				if script := world.GetScriptFuncs(entityID1); script.ContactRemoved != nil {
					script.ContactRemoved(world, entityID1, entityID2, updateParams)
				}
				if script := world.GetScriptFuncs(entityID2); script.ContactRemoved != nil {
					script.ContactRemoved(world, entityID2, entityID1, updateParams)
				}
			}
		}
	}

	world.processUpdates(updateParams)
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
