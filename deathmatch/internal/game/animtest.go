package game

import (
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type Animtest struct {
	Entity ecs.ID
}

func (Animtest) entity() {}

func (animtest Animtest) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	animation := animation("testcharacter4/animations/metarigAction")

	_ = animation

	skelly := skeleton("testcharacter4/skeletons/metarig")

	_ = skelly

	localTransforms := map[string]gmath.Affine3f32{}

	for _, bone := range []string{
		"upper_arm.L",
		"forearm.L",
		"hand.L",
	} {
		t := int(w.Now.Sub(0) / 1e8 % 30)

		localTransforms[bone] = gmath.TRHS3f32{
			R: gmath.Rot3{
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[1]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[2]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[3]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[0]", t),
			},
			S: gmath.Shcale3One(),
		}.ToAffine()
	}

	pose := Pose{
		Bones: map[string]gmath.Affine3f32{},
	}

	// TODO: this should probably be a method on Pose.
	for bone := range skelly.BindPose {
		A := gmath.Affine3One[float32]()

		tmp := bone
		for {
			B, ok := localTransforms[tmp]
			if !ok {
				B = gmath.Affine3One[float32]()
			}

			A = skelly.ParentRelative[tmp].Mul(B).Mul(A)

			parent, hasParent := skelly.Parent[tmp]
			if !hasParent {
				break
			}
			tmp = parent
		}

		pose.Bones[bone] = A.Mul(skelly.BindPoseInverse[bone])
	}

	w.Pose.Set(ourID, pose)

	relativeToBase := pose.Bones["hand.L"].Mul(skelly.BindPose["hand.L"])

	// TODO: use Get/SetGlobalTRS

	pos, _ := w.TranslationRotation.Get(ourID)

	// In reality we'll have support for parenting to bone in GetGlobalTRS
	w.TranslationRotation.Set(animtest.Entity, TranslationRotation{
		Translation: pos.Translation.Add(gmath.Vec3Convert[float64](relativeToBase.T)),
		Rotation: gmath.Rot3{
			0,
			0,
			1,
			0,
		},
	})
}
