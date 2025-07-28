package worldspawn

//go:generate stringer -trimprefix PhysicsShape -type PhysicsShapeKind -output physics_shape_kind_string.go

import (
	"errors"
	"fmt"
	"log"

	"worldspawn/geometry-go"
	"worldspawn/physics"
)

// TODO: rename PhysicsShape to CollisionShape
type PhysicsShapeKind int

const (
	_ PhysicsShapeKind = iota
	PhysicsShapeSphere
	PhysicsShapeBox
	PhysicsShapeCylinder
	PhysicsShapeFileBacked
)

// TODO: generate this the way stringer does. We may need to write our own
// generator for this
var physicsShapeKindFromString = map[string]PhysicsShapeKind{
	"Sphere":     PhysicsShapeSphere,
	"Box":        PhysicsShapeBox,
	"Cylinder":   PhysicsShapeCylinder,
	"FileBacked": PhysicsShapeFileBacked,
}

func (shapeKind PhysicsShapeKind) MarshalText() ([]byte, error) {
	return []byte(shapeKind.String()), nil
}

func (shapeKind *PhysicsShapeKind) UnmarshalText(text []byte) error {
	tmp, ok := physicsShapeKindFromString[string(text)]
	if !ok {
		return errors.New("unknown shape type")
	}
	*shapeKind = tmp
	return nil
}

// TODO: use the same container for both renderer and physics models
type CollisionGeometry struct {
	// TODO: remove these?
	Translation geometry.Vec3
	Rotation    geometry.Rot3
	Scale       geometry.Vec3

	Kind         PhysicsShapeKind
	Filename     string // used by Kind=FileBacked
	HalfExtent   geometry.Vec3
	ConvexRadius float32
}

/*
type ShapeCacheKey struct {
	Type int
	Filename string
	Radius float32
	HalfHeight float32
	Scale geometry.Vec3
}
*/

var shapeCache = make(map[CollisionGeometry]*physics.Shape)

func getShape(key CollisionGeometry) *physics.Shape {
	// TODO: canonicalize PhysicsShape, e.g. set HalfExtent to zero as necessary, etc

	// temp hack
	if key.Rotation == (geometry.Rot3{}) {
		key.Rotation = geometry.Rot3One()
	}
	if key.Scale == (geometry.Vec3{}) {
		key.Scale = geometry.Vec3Broadcast(1)
	}

	shape, ok := shapeCache[key]
	if ok {
		return shape
	}

	var err error
	switch key.Kind {
	case PhysicsShapeSphere:
		shape, err = physics.NewSphereShape(key.HalfExtent[0])

	case PhysicsShapeBox:
		shape, err = physics.NewBoxShape(key.HalfExtent, key.ConvexRadius)

	case PhysicsShapeCylinder:
		shape, err = physics.NewCylinderShape(key.HalfExtent[0], key.HalfExtent[2], key.ConvexRadius)

	case PhysicsShapeFileBacked:
		shape, err = physics.NewFileBackedShape(Data, key.Filename)

	default:
		panic(fmt.Sprintf("unknown physics shape kind %v", key.Kind))
	}
	if err != nil {
		// TODO: actually print a warning and return a box
		log.Fatal(err)
	}
	shape, err = physics.NewTransformedShape(key.Translation, key.Rotation, key.Scale, shape)
	if err != nil {
		// TODO: actually print a warning and return a box
		log.Fatal(err)
	}
	shapeCache[key] = shape
	return shape
}
