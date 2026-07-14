package game

import (
	"reflect"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/gmath"
)

type Animtest struct {
	Animation string
}

func init() {
	Scripts[reflect.TypeFor[Animtest]()] = script{
		Think: func(info *UpdateParams, world *World, entity Entity2, io IO) {
			io.EnqueueEntityUpdate(entity,
				func(info *UpdateParams, entity Entity2, io IO) {
					animtest := entity.ScriptState().(Animtest)

					animation := animation(animtest.Animation)

					skelly := entity.world.GetSkeleton(entity.ID())

					localTransforms := map[int]gmath.Affine3f32{}

					// localTransforms[skelly.JointByName("spine")] =
					// 	gmath.TRS3f32{
					// 		R: gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{1, 0, 0}).
					// 			Pow(float32(math.Sin(float64(w.Now.Sub(0)) / 1e9))),
					// 		S: gmath.Mat3x3UOne[float32](),
					// 	}.Compose()

					for _, bone := range []string{
						"upper_arm.L",
						"forearm.L",
						"hand.L",
					} {
						t := float64(info.Now.Sub(Time{})%1e9) / 1e9 * 30

						localTransforms[skelly.JointByName(bone)] =
							gmath.TRS3f32{
								R: gmath.Rot3{
									animation.Channels["pose.bones[\""+bone+"\"].rotation_quaternion[1]"].Sample(t),
									animation.Channels["pose.bones[\""+bone+"\"].rotation_quaternion[2]"].Sample(t),
									animation.Channels["pose.bones[\""+bone+"\"].rotation_quaternion[3]"].Sample(t),
									animation.Channels["pose.bones[\""+bone+"\"].rotation_quaternion[0]"].Sample(t),
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

					entity.SetPose(pose)
				})
		},
	}
}
