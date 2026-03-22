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

	localTransforms := map[string]gmath.Mat4x4{}

	for _, bone := range []string{
		"upper_arm.L",
		"forearm.L",
		"hand.L",
	} {
		t := int(w.Now.Sub(0) / 1e8 % 30)

		localTransforms[bone] = gmath.TRS3{
			R: gmath.Rot3{
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[1]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[2]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[3]", t),
				animation.Sample("pose.bones[\""+bone+"\"].rotation_quaternion[0]", t),
			},
			S: gmath.Vec3Ones(),
		}.Mat4x4()
	}

	pose := Pose{
		Bones: map[string]gmath.Mat4x4{},
	}

	// TODO: this should probably be a method on Pose.
	for bone := range skelly.BindPose {
		A := gmath.Mat4x4One()

		tmp := bone
		for {
			B, ok := localTransforms[tmp]
			if !ok {
				B = gmath.Mat4x4One()
			}

			A = skelly.ParentRelative[tmp].Mul4x4(B).Mul4x4(A)

			parent, hasParent := skelly.Parent[tmp]
			if !hasParent {
				break
			}
			tmp = parent
		}

		pose.Bones[bone] = A.Mul4x4(skelly.BindPoseInverse[bone])
	}

	w.Pose.Set(ourID, pose)

	relativeToBase := pose.Bones["hand.L"].Mul4x4(skelly.BindPose["hand.L"])

	// TODO: use Get/SetGlobalTRS

	pos, _ := w.TranslationRotation.Get(ourID)

	// In reality we'll have support for parenting to bone in GetGlobalTRS
	w.TranslationRotation.Set(animtest.Entity, TranslationRotation{
		Translation: pos.Translation.Add(gmath.Vec3Convert[float64](gmath.Vec3{
			relativeToBase[0][3],
			relativeToBase[1][3],
			relativeToBase[2][3],
		})),
		Rotation: gmath.Rot3{
			0,
			0,
			1,
			0,
		},
	})
}
