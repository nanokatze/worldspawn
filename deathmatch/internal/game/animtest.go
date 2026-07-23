package game

import (
	"reflect"
	"regexp"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
)

type Animtest struct {
	Animation unique.Handle[string]
}

func init() {
	Scripts[reflect.TypeFor[Animtest]()] = script{
		Think: func(info *UpdateParams, world *World, entity Entity2, io IO) {
			io.Update(entity,
				func(info *UpdateParams, entity Entity2, io IO) {
					animtest := entity.ScriptState().(Animtest)

					animation := animationCache.Get(animtest.Animation)

					sk := skeletonCache.Get(entity.Skeleton())

					t := float64(info.Now.Sub(Time{})%1e9) / 1e9 * 30

					localTransforms := make([]gmath.Affine3f32, sk.NumJoints())

					animatePose(animation, sk, localTransforms, t)

					pose := make(skeleton.Pose, sk.NumJoints())
					sk.ForwardKinematics(localTransforms, pose)

					entity.SetPose(pose)
				})
		},
	}
}

// TODO: don't use regexp but an actual parser please!!!
var posekey = regexp.MustCompile(`^pose\.bones\["([^"]*)"\]\.(.*)$`)

// TODO: instead of this function, we should have a function to apply []float32
// that (*animation.Animation).Sample spits out, to pose.
func animatePose(a *animation.Animation, sk *skeleton.Skeleton, pose []gmath.Affine3f32, t float64) {
	point := make([]float32, len(a.Channels()))
	a.Sample(t, point)

	shadow := make([]gmath.TRS3f32, len(pose))
	for i := range shadow {
		shadow[i] = gmath.TRS3One[float32]()
	}

	for i, ch := range a.Channels() {
		match := posekey.FindStringSubmatch(ch.Value())

		joint := &shadow[sk.JointByName(unique.Make(match[1]))]
		value := point[i]
		field := match[2]

		switch field {
		case "location[0]":
			joint.T[0] = value
		case "location[1]":
			joint.T[1] = value
		case "location[2]":
			joint.T[2] = value
		case "rotation_quaternion[0]":
			joint.R[3] = value
		case "rotation_quaternion[1]":
			joint.R[0] = value
		case "rotation_quaternion[2]":
			joint.R[1] = value
		case "rotation_quaternion[3]":
			joint.R[2] = value
		}
	}

	for i := range pose {
		pose[i] = shadow[i].Compose()
	}
}
