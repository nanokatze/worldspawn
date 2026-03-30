package animgraph

import "worldspawn/internal/gmath"

/*
type Skeleton struct {
	// jointNames []string
	// jointByName map[string]int

	parent          []int
	bindPose        []gmath.Affine3f32
	bindPoseInverse []gmath.Affine3f32
}
*/

type Skeleton struct {
	// Joints          []string

	// TODO: switch to a plain array with a string map for lookups

	Parent          map[string]string
	BindPose        map[string]gmath.Affine3f32
	BindPoseInverse map[string]gmath.Affine3f32
	ParentRelative  map[string]gmath.Affine3f32

	// other stuff
}
