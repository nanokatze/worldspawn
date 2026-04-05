package animgraph

import "worldspawn/internal/gmath"

// TODO: stick various methods onto the Pose object instead of exposing the
// internals. We want to be able to get relative to rest and absolute joint
// transforms.
// TODO: alternatively just kill this object?
type Pose struct {
	Bones map[int]gmath.Affine3f32
}
