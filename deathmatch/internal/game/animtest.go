package game

import (
	"maps"
	"reflect"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/cache"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
)

type Animtest struct {
	Animation unique.Handle[string]

	PlayTime Time
}

func init() {
	Scripts[reflect.TypeFor[Animtest]()] = script{
		Think: func(stx ScriptContext, world *World, entity Entity) {
			stx.Update(entity, func(stx ScriptContext, entity Entity) {
				state := entity.ScriptState().(Animtest)

				anim := animationCache.Get(state.Animation)

				sk := skeletonCache.Get(entity.Skeleton())

				point := make([]float32, len(anim.Channels()))
				animation.SampleTime(anim, stx.Now.Sub(state.PlayTime), point)

				localTransforms := make([]gmath.Affine3f32, sk.NumJoints())

				poseAnimCache.Get(poseAnimCacheKey{animation, sk})(point, localTransforms)

				pose := make(skeleton.Pose, sk.NumJoints())
				sk.ForwardKinematics(localTransforms, pose)

				entity.SetPose(pose)
			})
		},
	}
}

type poseAnimCacheKey struct {
	A  *animation.Animation
	Sk *skeleton.Skeleton
}

var poseAnimCache = cache.New(func(key poseAnimCacheKey) func(point []float32, pose []gmath.Affine3f32) {
	return poseAnimator(key.A, key.Sk)
})

// TODO: move this to internal/animation
func poseAnimator(a *animation.Animation, sk *skeleton.Skeleton) func(point []float32, pose []gmath.Affine3f32) {
	type trs3Chmap struct {
		T [3]int
		R [4]int
		// S [6]int
	}

	channels := maps.Collect(func(yield func(string, int) bool) {
		for i, name := range a.Channels() {
			yield(name, i)
		}
	})

	lookupchannel := func(name string) int {
		if index, ok := channels[name]; ok {
			return index
		}
		return -1
	}

	chmap := make([]trs3Chmap, sk.NumJoints())
	// TODO: should we loop over channels or over joints?
	for i := range sk.NumJoints() {
		// TODO: skip if a particular joint is not animated. We could also make
		// things sparse.

		// TODO: we should have our own channel names. They should have a form
		// of entitybinding.field.subfield..., Though in certain cases I suppose
		// we'll need to add magic names, such as with the pose.
		chmap[i] = trs3Chmap{
			T: [3]int{
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].location[0]"),
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].location[1]"),
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].location[2]"),
			},
			R: [4]int{
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].rotation_quaternion[1]"),
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].rotation_quaternion[2]"),
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].rotation_quaternion[3]"),
				lookupchannel("pose.bones[\"" + sk.JointNames[i].Value() + "\"].rotation_quaternion[0]"),
			},
		}
	}

	return func(point []float32, pose []gmath.Affine3f32) {
		if len(pose) != len(chmap) {
			panic("guh")
		}

		for i, j := range chmap {
			// TODO: decompose if things are partially animated
			tmp := gmath.TRS3One[float32]()

			if j.T[0] != -1 {
				tmp.T[0] = point[j.T[0]]
			}
			if j.T[1] != -1 {
				tmp.T[1] = point[j.T[1]]
			}
			if j.T[2] != -1 {
				tmp.T[2] = point[j.T[2]]
			}

			if j.R[0] != -1 {
				tmp.R[0] = point[j.R[0]]
			}
			if j.R[1] != -1 {
				tmp.R[1] = point[j.R[1]]
			}
			if j.R[2] != -1 {
				tmp.R[2] = point[j.R[2]]
			}
			if j.R[3] != -1 {
				tmp.R[3] = point[j.R[3]]
			}
			tmp.R = tmp.R.Renormalize()

			pose[i] = tmp.Affine()
		}
	}
}
