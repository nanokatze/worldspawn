package game

// TODO: eventually kill this off in favor of some geometry nodes -esque
// mechanism (https://github.com/nanokatze/worldspawn-private/issues/45)

import (
	"worldspawn/internal/gmath"
)

type sphere struct {
	Radius float32
}

type cylinder struct {
	Radius       float32
	Height       float32
	ConvexRadius float32
}

type fileBackedGeometry struct {
	Filename string
}

type transformedShape struct {
	Translation gmath.Vec3f32
	Rotation    gmath.Rot3
	Shape       shape // TODO: this should be type shape (pair of arbitrary and convex)
}
