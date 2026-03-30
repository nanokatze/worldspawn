package animgraph

import "worldspawn/internal/gmath"

// TODO: make the internals private probably?
type Skeleton struct {
	JointNames   []string
	JointByName_ map[string]int

	Parent          []int
	BindPose        []gmath.Affine3f32
	BindPoseInverse []gmath.Affine3f32
	ParentRelative  []gmath.Affine3f32
}

func (s *Skeleton) JointByName(name string) int {
	if i, ok := s.JointByName_[name]; ok {
		return i
	}
	return -1
}
