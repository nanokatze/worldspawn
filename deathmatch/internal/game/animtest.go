package game

import (
	"reflect"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
)

// TODO: factor things out into an actual AnimationPlayer "component" (component
// as in UE)
type Animtest struct {
	Animation unique.Handle[string]

	PlayTime Time
}

func init() {
	Scripts[reflect.TypeFor[Animtest]()] = script{
		Think: func(stx ScriptContext, entity Entity, world *World) {
			stx.Update(entity, func(stx ScriptContext, entity Entity) {
				state := entity.ScriptState().(Animtest)

				anim := animationCache.Get(state.Animation)

				sk := skeletonCache.Get(entity.Skeleton())

				v := make([]float32, len(anim.Channels()))
				animation.SampleTime(anim, stx.Now.Sub(state.PlayTime), v)

				localTransforms := make([]gmath.Affine3f32, sk.NumJoints())

				poseMapperCache.Get(poseMapperKey{anim, sk})(v, localTransforms)

				pose := make(skeleton.Pose, sk.NumJoints())
				sk.ForwardKinematics(localTransforms, pose)

				entity.SetPose(pose)
			})
		},
	}
}
