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

func (e Entity2) Velocity() Velocity { return e.world.Velocity.Load(e.id.Index()) }

func (e Entity2) SetVelocity(v Velocity) { e.world.Velocity.Store(e.id.Index(), v) }

// Always run this before performing physics queries!!!
//
// TODO: move out of this file kinda? I'm thinking into step.go
func (world *World) updatePhysicsShadow(updateParams *UpdateParams) {
	// TODO: remove bodies when we delete entities!!!!!!!!
	for id := range ecs.All(&world.physicsBodyExists) {
		if _, ok := world.CollisionLayer.Get(id); !ok {
			world.physics.RemoveBody(physics.BodyID(id))
			world.physicsBodyExists.Delete(id)
		}
	}

	for id, layer := range ecs.All(&world.CollisionLayer) {
		tr, _ := world.TransformTR.Get(id)
		// TODO: ensure trs.S is 1
		velocity, _ := world.Velocity.Get(id)
		_, sensor := world.CollisionSensor.Get(id)

		motionType2 := collisionLayerMotionType[layer]

		gravityFactor, ok := world.GravityFactor.Get(id)
		if !ok {
			gravityFactor = 1
		}

		shape2 := getShape(Entity2{world, id})

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
				tr.T,
				tr.R,
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
				tr.T,
				tr.R,
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

// TODO: procedural shape functions should use a more limited interface than
// Entity2 directly. On the other hand, I'm not so sure how we would allow these
// functions to perform collision queries and such then.
var proceduralShapes = map[string]func(entity Entity2) shape{
	"Grenade": func(entity Entity2) shape {
		return getShape2(sphere{
			Radius: 0.0568,
		})
	},
	"Gladiator": func(entity Entity2) shape {
		return getShape2(transformedShape{
			Translation: gmath.Vec3f32{0, 0, 1.9 / 2}, // TODO: read standing height off entity
			Rotation:    gmath.Rot3One(),

			Shape: getShape2(cylinder{
				Radius: 0.4,
				Height: 1.9,
			}),
		})
	},
}

func getShape(entity Entity2) *physics.Shape {
	// TOOD: introduce proper accessors for these?
	layer := entity.world.CollisionLayer.Load(entity.id.Index())
	geometry := entity.world.CollisionGeometry.Load(entity.id.Index())

	motionType2 := collisionLayerMotionType[layer]

	// HACK: our gross way out of not having geonodes, see
	// https://github.com/nanokatze/worldspawn-private/issues/45
	//
	// TODO: this should probably be handled in getShape2 directly. In the
	// future we'll allow any file-backed geometry to be procedural (and not
	// just the top level). We'll have to make caching work in such a way that
	// procedural bits are re-evaluated all the time, but the constant bits
	// which need extra processing to construct are cached.
	var shape shape
	if shapeFunc, ok := proceduralShapes[geometry.Value()]; ok {
		shape = shapeFunc(entity)
	} else {
		shape = getShape2(fileBackedGeometry{geometry.Value()})
	}

	if motionType2 != 0 && shape.convex != nil {
		return shape.convex
	}
	return shape.shape
}

func (world *World) physicsStep(updateParams *UpdateParams) {
	// TODO: push it back onto the user again?
	world.updatePhysicsShadow(updateParams)

	world.physics.Update(float32(durationToFloatSeconds(updateParams.Δt)), updateParams.Gravity)

	for _, bodyID := range world.physics.ActiveBodies() {
		entityID := ecs.ID(bodyID) // BUG: this is not correct anymore because the generations do not match!!!
		entity := world.GetEntity2(entityID)

		pos, rot, linVel, angVel := world.physics.WritebackBody(bodyID)

		entity.SetTranslationAndRotation(TR3f64{T: pos, R: rot})

		// TODO: don't store velocity back for kinematic bodies
		entity.SetVelocity(Velocity{Linear: linVel, Angular: angVel})
	}

	for _, ce := range world.physics.ContactEvents() {
		// TODO: properly translate bodyID to entityID
		entityID1 := ecs.ID(ce.Body1.BodyID)
		entityID2 := ecs.ID(ce.Body2.BodyID)

		// TODO: spawn sounds on collision

		// One or both entities in the contact event might've been removed
		// before the last Update

		entity1 := world.GetEntity2(entityID1)
		entity2 := world.GetEntity2(entityID2)
		if entity1.Valid() && entity2.Valid() {
			switch ce.Type {
			case 1:
				if script := entity1.Script(); script.ContactAdded != nil {
					script.ContactAdded(updateParams, world, entityID1, entityID2)
				}
				if script := entity2.Script(); script.ContactAdded != nil {
					script.ContactAdded(updateParams, world, entityID2, entityID1)
				}

			case 2:
				if script := entity1.Script(); script.ContactRemoved != nil {
					script.ContactRemoved(updateParams, world, entityID1, entityID2)
				}
				if script := entity2.Script(); script.ContactRemoved != nil {
					script.ContactRemoved(updateParams, world, entityID2, entityID1)
				}
			}
		}
	}

	world.processUpdates(updateParams)
}

type shape struct {
	shape  *physics.Shape
	convex *physics.Shape
}

// TODO: make this weak-valued sync.Map
var shapeCache = make(map[any]shape)

func getShape2(key any) shape {
	shape, ok := shapeCache[key]
	if ok {
		return shape
	}

	var err error
	switch key := key.(type) {
	case sphere:
		shape.shape, err = physics.NewSphereShape(key.Radius)
		if err != nil {
			panic(err)
		}
	// case geometryBox:
	// 	shape.shape, err = physics.NewBoxShape(key.HalfExtent, key.ConvexRadius)
	case cylinder:
		shape.shape, err = physics.NewCylinderShape(key.Radius, key.Height/2, key.ConvexRadius)
		if err != nil {
			panic(err)
		}
	case fileBackedGeometry:
		shape.shape, err = physics.NewFileBackedShape(Data, key.Filename, true)
		if err != nil {
			panic(err)
		}
		shape.convex, err = physics.NewFileBackedShape(Data, key.Filename, false)
		if err != nil {
			panic(err)
		}
	case transformedShape:
		shape.shape, err = physics.NewTransformedShape(key.Translation, key.Rotation, gmath.Vec3Ones[float32](), key.Shape.shape)
		if err != nil {
			panic(err)
		}
		if key.Shape.convex != nil {
			shape.convex, err = physics.NewTransformedShape(key.Translation, key.Rotation, gmath.Vec3Ones[float32](), key.Shape.convex)
			if err != nil {
				panic(err)
			}
		}
	default:
		panic(fmt.Sprintf("bad physics shape desc %#v", key))
	}

	shapeCache[key] = shape
	return shape
}
