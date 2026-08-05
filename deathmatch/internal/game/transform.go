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

func (e Entity) SetParent(v ecs.ID) { e.world.SetParent(e.id, v) }

func (e Entity) SetParentBone(v unique.Handle[string]) { e.world.ParentBone.Store(e.id.Index(), v) }

func (e Entity) Transform() gmath.TRS3f64 {
	// TODO: validate that the transform is invertible? We might wanna ban non-invertible transforms

	tr := e.world.TransformTR.Load(e.id.Index())
	s := e.world.TransformS.Load(e.id.Index())
	return gmath.TRS3f64{tr.T, tr.R, s}
}

func (e Entity) SetTransform(v gmath.TRS3f64) {
	// TODO: validate the transform

	e.world.TransformTR.Store(e.id.Index(), TR3f64{v.T, v.R})
	e.world.TransformS.Store(e.id.Index(), v.S)
}

func (e Entity) SetTransformTR(v TR3f64) { e.world.TransformTR.Store(e.id.Index(), v) }

// TODO: if we encounter errors during hierarchy traversal we should restart
// traversal with diagnostics collection and print the collected diagnostics
// after using World.logger.Error
//
// TODO: cycle detection
//
// TODO: replace T.Mul(A) with just A on the first iteration to optimize the
// common case
func (world *World) GetGlobalTransform2(entity Entity) gmath.Affine3f64 {
	getEntityTransform := func(entity Entity) gmath.Affine3f64 {
		return entity.Transform().Compose()
	}

	getBoneTransform := func(entity Entity, bone unique.Handle[string]) gmath.Affine3f32 {
		sk := skeletonCache.Get(entity.Skeleton())
		if sk == nil {
			return gmath.Affine3One[float32]()
		}
		pose := entity.Pose()
		return pose.Get(sk.JointByName(bone), sk)
	}

	// TODO: don't hardcode the hierarchy depth bound
	// TODO: actually maybe have a bloom filter/small hashmap to track cycles?
	// It would be nice to avoid having a different behavior regardless of
	// whether we have cycle detection on or not.
	//
	// NOTE: the hierarchy depth is bounded by no. of entries in the table

	A := gmath.Affine3One[float64]()
	if !entity.IsValid() {
		return A
	}
	for range 5000 {
		A = getEntityTransform(entity).Mul(A)

		parent := world.Entity(entity.Parent())
		if !parent.IsValid() {
			// TODO: ensure that parent to bone isn't set in this case
			break
		}

		if parentBone := entity.ParentBone(); parentBone != (unique.Handle[string]{}) {
			A = getBoneTransform(parent, parentBone).Convert[float64]().Mul(A)
		}

		entity = parent
	}

	return A
}

func (world *World) GetGlobalTransform(id ecs.ID) gmath.Affine3f64 {
	return world.GetGlobalTransform2(world.Entity(id))
}
