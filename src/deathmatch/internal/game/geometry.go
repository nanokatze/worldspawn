package game

//go:generate stringer -trimprefix Geometry -type GeometryKind -output geometry_kind_string.go

import (
	"errors"

	"worldspawn/geometry-go"
	"worldspawn/internal/nice"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type GeometryKind int

const (
	_ GeometryKind = iota
	GeometrySphere
	GeometryBox
	GeometryCylinder
	GeometryFileBacked
)

var collisionGeometryKindFromString = map[string]GeometryKind{
	"Sphere":     GeometrySphere,
	"Box":        GeometryBox,
	"Cylinder":   GeometryCylinder,
	"FileBacked": GeometryFileBacked,
}

func (shapeKind *GeometryKind) MarshalText() ([]byte, error) {
	return []byte(shapeKind.String()), nil
}

func (shapeKind *GeometryKind) UnmarshalText(text []byte) error {
	tmp, ok := collisionGeometryKindFromString[string(text)]
	if !ok {
		return errors.New("unknown shape type")
	}
	*shapeKind = tmp
	return nil
}

// TODO: make it an interface with various implementations (FileBacked, etc.)
type Geometry struct {
	// TODO: remove the transform in favor of an option to use children entities
	// as a way to specify compound geometry.
	Translation geometry.Vec3
	Rotation    geometry.Rot3
	Scale       geometry.Vec3

	Kind         GeometryKind
	Filename     string // used by Kind=FileBacked
	HalfExtent   geometry.Vec3
	ConvexRadius float32
}

type GeometryPacked string

func PackGeometry(geo Geometry) GeometryPacked {
	// TODO: ugh
	if geo.Rotation == (geometry.Rot3{}) {
		geo.Rotation = geometry.Rot3One()
	}
	if geo.Scale == (geometry.Vec3{}) {
		geo.Scale = geometry.Vec3Broadcast(1)
	}

	buf, err := nice.Marshal(&geo)
	if err != nil {
		panic(err)
	}
	return GeometryPacked(buf)
}

func UnpackGeometry(packed GeometryPacked) Geometry {
	var unpacked Geometry
	if err := nice.Unmarshal([]byte(packed), &unpacked); err != nil {
		panic(err)
	}
	return unpacked
}

func (geo *GeometryPacked) UnmarshalJSONFrom(d *jsontext.Decoder) error {
	var tmp Geometry
	if err := json.UnmarshalDecode(d, &tmp); err != nil {
		return err
	}
	*geo = PackGeometry(tmp)
	return nil
}
