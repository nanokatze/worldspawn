package game

import (
	"worldspawn/internal/animgraph"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type Animtest struct {
}

func (Animtest) entity() {}

func (scene *Scene) GetSkeleton(id ecs.ID) *animgraph.Skeleton {
	skellyName, ok := scene.Skeleton.Get(id)
	if !ok {
		return nil
	}
	return skeleton(skellyName)
}

func (animtest Animtest) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	animation := animation("testcharacter4/animations/metarigAction")

	skelly := w.GetSkeleton(ourID)

	localTransforms := map[int]gmath.Affine3f32{}

	for _, bone := range []string{
		"upper_arm.L",
		"forearm.L",
		"hand.L",
	} {
		t := float64(w.Now.Sub(0)%1e9) / 1e9 * 30

		localTransforms[skelly.JointByName(bone)] =
			gmath.TRS3f32{
				R: gmath.Rot3{
					animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[1]", t),
					animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[2]", t),
					animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[3]", t),
					animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[0]", t),
				}.Renormalize(),
				S: gmath.Mat3x3UOne[float32](),
			}.Compose()
	}

	pose := animgraph.Pose{
		Bones: map[int]gmath.Affine3f32{},
	}

	// TODO: flooding would be more efficient
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
