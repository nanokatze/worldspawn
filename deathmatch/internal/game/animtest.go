package game

import (
	"worldspawn/internal/animgraph"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type Animtest struct {
}

func (Animtest) entity() {}

func (animtest Animtest) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	animation := animation("testcharacter4/animations/metarigAction")

	_ = animation

	skelly := skeleton("testcharacter4/skeletons/metarig")

	_ = skelly

	localTransforms := map[int]gmath.Affine3f32{}

	for _, bone := range []string{
		"upper_arm.L",
		"forearm.L",
		"hand.L",
	} {
		t := int(w.Now.Sub(0) / 1e8 % 30)

		localTransforms[skelly.JointByName(bone)] = gmath.TRS3f32{
			R: gmath.Rot3{
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[1]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[2]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[3]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[0]", t),
			},
			S: gmath.Mat3x3UOne[float32](),
		}.Compose()
	}

	pose := animgraph.Pose{
		Skelly: skelly,
		Bones:  map[int]gmath.Affine3f32{},
	}

	// TODO: this should probably be a method on Pose.
	for bone := range skelly.JointNames {
		A := gmath.Affine3One[float32]()

		tmp := bone
		for {
			B, ok := localTransforms[tmp]
			if !ok {
				B = gmath.Affine3One[float32]()
			}

			A = skelly.ParentRelative[tmp].Mul(B).Mul(A)

			parent := skelly.Parent[tmp]
			if parent == -1 {
				break
			}
			tmp = parent
		}

		pose.Bones[bone] = A.Mul(skelly.BindPoseInverse[bone])
	}

	w.Pose.Set(ourID, pose)
}
