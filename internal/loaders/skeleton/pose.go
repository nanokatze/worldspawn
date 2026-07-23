package skeleton

import "worldspawn/internal/gmath"

// TODO: mention the representation in the type name
// TODO: stick *Skeleton into the Pose?
type Pose []gmath.Affine3f32

func (pose Pose) Validate(skeleton *Skeleton) bool {
	return len(pose) == skeleton.NumJoints()
}

// TODO: not sure if this should be a Pose or Skeleton method. I guess it's more
// appropriate for this to be a Pose method because Skeleton doesn't really need
// to be aware of a concept of a Pose.
func (pose Pose) Get(i int, skeleton *Skeleton) gmath.Affine3f32 {
	if !pose.Validate(skeleton) {
		return gmath.Affine3One[float32]()
	}
	if !(0 <= i && i < len(pose)) {
		return gmath.Affine3One[float32]()
	}
	return pose[i].Mul(skeleton.BindPose[i])
}
