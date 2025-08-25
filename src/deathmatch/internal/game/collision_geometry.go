package game

//go:generate stringer -trimprefix CollisionGeometry -type CollisionGeometryKind -output collision_geometry_kind_string.go

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"worldspawn/geometry-go"
	"worldspawn/internal/nice"
	"worldspawn/physics"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type CollisionGeometryKind int

const (
	_ CollisionGeometryKind = iota
	CollisionGeometrySphere
	CollisionGeometryBox
	CollisionGeometryCylinder
	CollisionGeometryFileBacked
)

var collisionGeometryKindFromString = map[string]CollisionGeometryKind{
	"Sphere":     CollisionGeometrySphere,
	"Box":        CollisionGeometryBox,
	"Cylinder":   CollisionGeometryCylinder,
	"FileBacked": CollisionGeometryFileBacked,
}

func (shapeKind CollisionGeometryKind) MarshalText() ([]byte, error) {
	return []byte(shapeKind.String()), nil
}

func (shapeKind *CollisionGeometryKind) UnmarshalText(text []byte) error {
	tmp, ok := collisionGeometryKindFromString[string(text)]
	if !ok {
		return errors.New("unknown shape type")
	}
	*shapeKind = tmp
	return nil
}

// TODO: make it an interface with various implementations (FileBacked, etc)
type CollisionGeometry struct {
	// TODO: remove these?
	Translation geometry.Vec3
	Rotation    geometry.Rot3
	Scale       geometry.Vec3

	Kind         CollisionGeometryKind
	Filename     string // used by Kind=FileBacked
	HalfExtent   geometry.Vec3
	ConvexRadius float32
}

type CollisionGeometry2 string

// TODO: rename
func (tmp *CollisionGeometry) Unpack(col CollisionGeometry2) {
	if err := nice.UnmarshalDecode(nice.NewDecoder(strings.NewReader(string(col))), tmp); err != nil {
		panic(err)
	}
}

// TODO: rename
func (tmp CollisionGeometry) Pack() CollisionGeometry2 {
	// TODO: ugh
	if tmp.Rotation == (geometry.Rot3{}) {
		tmp.Rotation = geometry.Rot3One()
	}
	if tmp.Scale == (geometry.Vec3{}) {
		tmp.Scale = geometry.Vec3Broadcast(1)
	}

	var buf strings.Builder
	if err := nice.MarshalEncode(nice.NewEncoder(&buf), &tmp); err != nil {
		panic(err)
	}
	return CollisionGeometry2(buf.String())
}

func (geo *CollisionGeometry2) UnmarshalJSONFrom(d *jsontext.Decoder) error {
	var tmp CollisionGeometry
	if err := json.UnmarshalDecode(d, &tmp); err != nil {
		return err
	}
	*geo = tmp.Pack()
	return nil
}

var shapeCache = make(map[CollisionGeometry2]*physics.Shape)

func getShape(key2 CollisionGeometry2) *physics.Shape {
	// TODO: canonicalize PhysicsShape, e.g. set HalfExtent to zero as necessary, etc

	shape, ok := shapeCache[key2]
	if ok {
		return shape
	}

	var key CollisionGeometry
	key.Unpack(key2)

	var err error
	switch key.Kind {
	case CollisionGeometrySphere:
		shape, err = physics.NewSphereShape(key.HalfExtent[0])

	case CollisionGeometryBox:
		shape, err = physics.NewBoxShape(key.HalfExtent, key.ConvexRadius)

	case CollisionGeometryCylinder:
		shape, err = physics.NewCylinderShape(key.HalfExtent[0], key.HalfExtent[2], key.ConvexRadius)

	case CollisionGeometryFileBacked:
		shape, err = physics.NewFileBackedShape(Data, key.Filename)

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
	shapeCache[key2] = shape
	return shape
}
