package animgraph

import "worldspawn/internal/gmath"

type Skeleton struct {
	// jointNames []string
	// jointByName map[string]int

	parent          []int
	bindPose        []gmath.Mat4x4 // TODO: change this to be Affine3
	bindPoseInverse []gmath.Mat4x4
}
