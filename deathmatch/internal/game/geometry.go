package game

// TODO: eventually kill this off in favor of some geometry nodes -esque
// mechanism (https://github.com/nanokatze/worldspawn-private/issues/45)

import (
	"worldspawn/internal/gmath"
	"worldspawn/internal/nice"
)

type geometryKind int

const (
	_ geometryKind = iota
	geometrySphere
	geometryBox
	geometryCylinder
	geometryFileBacked
)

// TODO: make it an interface with various implementations (FileBacked, etc.)
type _Geometry struct {
	// TODO: remove the transform in favor of an option to use children entities
	// as a way to specify compound geometry.
	Translation gmath.Vec3
	Rotation    gmath.Rot3
	Scale       gmath.Vec3

	Kind         geometryKind
	Filename     string // used by Kind=FileBacked
	HalfExtent   gmath.Vec3
	ConvexRadius float32
}

type geometryPacked string

func packGeometry(geo _Geometry) geometryPacked {
	// TODO: ugh
	if geo.Rotation == (gmath.Rot3{}) {
		geo.Rotation = gmath.Rot3One()
	}
	if geo.Scale == (gmath.Vec3{}) {
		geo.Scale = gmath.Vec3Ones[float32]()
	}

	buf, err := nice.Marshal(&geo)
	if err != nil {
		panic(err)
	}
	return geometryPacked(buf)
}

func unpackGeometry(packed geometryPacked) _Geometry {
	var unpacked _Geometry
	if err := nice.Unmarshal([]byte(packed), &unpacked); err != nil {
		panic(err)
	}
	return unpacked
}
