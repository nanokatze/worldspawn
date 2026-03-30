package animgraph

import "worldspawn/internal/gmath"

// TODO: stick various methods onto the Pose object instead of exposing the
// internals. We want to be able to get relative to rest and absolute joint
// transforms.
type Pose struct {
	// TODO: delegate passing *Skeleton to the user when bone position etc? That
	// doesn't seem very useful, but would let the user straightforwardly
	// reutilize storage. We might honestly just kill Pose in favor of
	// plain []gmath.Affine3f32 × *Skeleton.
	Skelly *Skeleton
	Bones  map[int]gmath.Affine3f32
}
