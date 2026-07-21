package skeleton

import "worldspawn/internal/gmath"

type Pose []gmath.Affine3f32

func (pose Pose) Validate(skeleton *Skeleton) bool {
	return len(pose) == skeleton.NumJoints()
}

func (pose Pose) Get(i int, skeleton *Skeleton) gmath.Affine3f32 {
	if !pose.Validate(skeleton) {
		return gmath.Affine3One[float32]()
	}
	if !(0 <= i && i < len(pose)) {
		return gmath.Affine3One[float32]()
	}
	return pose[i].Mul(skeleton.BindPose[i])
}
