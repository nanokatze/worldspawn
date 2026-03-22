package animgraph

import "worldspawn/internal/gmath"

// TODO: stick various methods onto the Pose object instead of exposing the
// internals. We want to be able to get relative to rest and absolute joint
// transforms.
type Pose struct {
	Bones map[string]gmath.Mat4x4 // TODO: change this to be Affine3
}
