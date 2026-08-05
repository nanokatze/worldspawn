package game

import (
	"maps"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/cache"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
)

func (e Entity) Parent() ecs.ID { return e.world.Parent.Load(e.id.Index()) }

func (e Entity) ParentBone() unique.Handle[string] { return e.world.ParentBone.Load(e.id.Index()) }

func (e Entity) Skeleton() unique.Handle[string] { return e.world.Skeleton.Load(e.id.Index()) }

func (e Entity) SetSkeleton(v unique.Handle[string]) { e.world.Skeleton.Store(e.id.Index(), v) }

func (e Entity) Pose() skeleton.Pose { return e.world.Columns.pose[e.id.Index()] }

// Note that pose is not replicated
//
// TODO: change up the api to encourage slice reuse
func (e Entity) SetPose(v skeleton.Pose) { e.world.Columns.pose[e.id.Index()] = v }

var animationCache = cache.New(func(key unique.Handle[string]) *animation.Animation {
	f, err := Data.Open(key.Value())
	if err != nil {
		panic(err)
	}
	defer f.Close()

	animation, err := animation.Read(f)
	if err != nil {
		panic(err)
	}

	return animation
})

var skeletonCache = cache.New(func(key unique.Handle[string]) *skeleton.Skeleton {
	f, err := Data.Open(key.Value())
	if err != nil {
		panic(err)
	}
	defer f.Close()

	skeleton, err := skeleton.Read(f)
	if err != nil {
		panic(err)
	}

	return skeleton
})

type poseMapperKey struct {
	A  *animation.Animation
	Sk *skeleton.Skeleton
}

var poseMapperCache = cache.New(func(key poseMapperKey) func(v []float32, pose []gmath.Affine3f32) {
	return makePoseMapper(key.A, key.Sk)
})

// TODO: move this to internal/animation
// TODO: swap value and pose places? I.e. pose as 0th arg, for dst.
func makePoseMapper(a *animation.Animation, sk *skeleton.Skeleton) func(v []float32, pose []gmath.Affine3f32) {
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

	return func(v []float32, pose []gmath.Affine3f32) {
		if len(pose) != len(chmap) {
			panic("guh")
		}

		for i, j := range chmap {
			// TODO: decompose if things are partially animated
			tmp := gmath.TRS3One[float32]()

			if j.T[0] != -1 {
				tmp.T[0] = v[j.T[0]]
			}
			if j.T[1] != -1 {
				tmp.T[1] = v[j.T[1]]
			}
			if j.T[2] != -1 {
				tmp.T[2] = v[j.T[2]]
			}

			if j.R[0] != -1 {
				tmp.R[0] = v[j.R[0]]
			}
			if j.R[1] != -1 {
				tmp.R[1] = v[j.R[1]]
			}
			if j.R[2] != -1 {
				tmp.R[2] = v[j.R[2]]
			}
			if j.R[3] != -1 {
				tmp.R[3] = v[j.R[3]]
			}
			tmp.R = tmp.R.Renormalize()

			pose[i] = tmp.Affine()
		}
	}
}
