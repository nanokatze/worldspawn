package animgraph

import (
	"unique"

	"worldspawn/internal/gmath"
)

// TODO: introduce skeleton builder to simplify loader code

type SkeletonBuilder struct {
}

// TODO: make the internals private probably?
type Skeleton struct {
	JointNames   []unique.Handle[string]
	JointByName_ map[unique.Handle[string]]int

	Parent          []int
	Children        [][]int
	BindPose        []gmath.Affine3f32
	BindPoseInverse []gmath.Affine3f32
	ParentRelative  []gmath.Affine3f32
}

func (s *Skeleton) JointByName(name unique.Handle[string]) int {
	if i, ok := s.JointByName_[name]; ok {
		return i
	}
	return -1
}
