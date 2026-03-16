package game

import (
	"math"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type Animtest struct {
	Entity ecs.ID
}

func (Animtest) entity() {}

func (animtest Animtest) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	skelly := skeleton("testcharacter4/skeletons/metarig")

	relativeToRest := skelly.BindPose["forearm.L"].Mul4x4(gmath.TRS3{
		R: gmath.Rot3InPlane(gmath.Vec3{0, 0, 1}, float32(math.Sin(float64(w.Now)/1e9*10))),
		S: gmath.Vec3Ones(),
	}.Mat4x4()).Mul4x4(skelly.BindPoseInverse["forearm.L"])

	w.Pose.Set(ourID, Pose{
		Bones: map[string]gmath.Mat4x4{
			"forearm.L": relativeToRest,
		},
	})

	// TODO: compute relative to base properly
	relativeToBase := relativeToRest

	// TODO: use Get/SetGlobalTRS

	pos, _ := w.TranslationRotation.Get(ourID)

	// In reality we'll have support for parenting to bone in GetGlobalTRS
	w.TranslationRotation.Set(animtest.Entity, TranslationRotation{
		Translation: pos.Translation.Add(gmath.Vec3Convert[float64](gmath.Vec3{
			relativeToBase[0][3],
			relativeToBase[1][3],
			relativeToBase[2][3],
		})),
		Rotation: gmath.Rot3One(),
	})
}
