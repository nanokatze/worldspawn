package game

import (
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type TR3f64 struct {
	T gmath.Vec3f64
	R gmath.Rot3
}

func (e Entity2) SetParent(v ecs.ID) { e.world.SetParent(e.id, v) }

func (e Entity2) SetParentBone(v unique.Handle[string]) { e.world.ParentBone.Store(e.id.Index(), v) }

func (e Entity2) Transform() gmath.TRS3f64 {
	// TODO: validate that the transform is invertible? We might wanna ban non-invertible transforms

	tr := e.world.TransformTR.Load(e.id.Index())
	s := e.world.TransformS.Load(e.id.Index())
	return gmath.TRS3f64{tr.T, tr.R, s}
}

func (e Entity2) SetTransform(v gmath.TRS3f64) {
	// TODO: validate the transform

	e.world.TransformTR.Store(e.id.Index(), TR3f64{v.T, v.R})
	e.world.TransformS.Store(e.id.Index(), v.S)
}

func (e Entity2) SetTranslationAndRotation(v TR3f64) { e.world.TransformTR.Store(e.id.Index(), v) }

func (world *World) GetGlobalTransform2(entity Entity2) gmath.Affine3f64 {
	return world.GetGlobalTransform(entity.ID())
}

// TODO: if we encounter errors during hierarchy traversal we should restart
// traversal with diagnostics collection and print the collected diagnostics
// after using Scene.Logger.Error
//
// TODO: cycle detection
//
// TODO: replace T.Mul(A) with just A on the first iteration to optimize the
// common case
//
// TODO: clean this up. Could Entity2 help here?
// TODO: remove this in favor of GetGlobalTransform2
func (world *World) GetGlobalTransform(id ecs.ID) gmath.Affine3f64 {
	getEntityTransform := func(id ecs.ID) gmath.Affine3f64 {
		tr, ok := world.TransformTR.Get(id)
		if !ok {
			return gmath.Affine3One[float64]()
		}
		s, ok := world.TransformS.Get(id)
		if !ok {
			s = gmath.Mat3x3UOne[float32]()
		}
		return gmath.TRS3f64{tr.T, tr.R, s}.Compose()
	}

	// TODO: make this a method on the scene?
	getBoneTransform := func(id ecs.ID, bone unique.Handle[string]) gmath.Affine3f32 {
		skelly := world.GetSkeleton(id)
		if skelly == nil {
			return gmath.Affine3One[float32]()
		}
		boneIndex := skelly.JointByName(bone)
		if boneIndex == -1 {
			return gmath.Affine3One[float32]()
		}

		pose := world.Entities.Pose[id.Index()]
		boneTransform, ok := pose.Bones[boneIndex]
		if !ok {
			return skelly.BindPose[boneIndex]
		}
		return boneTransform.Mul(skelly.BindPose[boneIndex])
	}

	// TODO: don't hardcode the hierarchy depth bound
	// TODO: actually maybe have a bloom filter/small hashmap to track cycles?
	// It would be nice to avoid having a different behavior regardless of
	// whether we have cycle detection on or not.
	//
	// NOTE: the hierarchy depth is bounded by no. of entries in the table

	A := gmath.Affine3One[float64]()
	for range 5000 {
		A = getEntityTransform(id).Mul(A)

		parent := world.GetParent(id)
		if parent == 0 {
			// TODO: ensure that parent to bone isn't set
			break
		}

		if parentBone, parentedToBone := world.ParentBone.Get(id); parentedToBone {
			A = getBoneTransform(parent, parentBone).Convert[float64]().Mul(A)
		}

		id = parent
	}

	return A
}
