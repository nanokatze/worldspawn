package geometry

import (
	"log"
	"math"
	"testing"

	"worldspawn/gpu"
	"worldspawn/internal/gmath"
)

func TestXxx(t *testing.T) {
	vertexCount := 4
	skinnedPositions := gpu.MakeSliceUncached[[3]float32](vertexCount)
	restPositions := gpu.MakeSliceUncached[[3]float32](vertexCount)

	jointsPerVertex := 1
	jointWeights := gpu.MakeSliceUncached[Uhh](vertexCount * jointsPerVertex)

	pose := gpu.MakeSliceUncached[gmath.Mat4x4](jointsPerVertex)

	for i := range vertexCount {
		restPositions.Value()[i] = gmath.Vec3{
			float32(math.Cos(float64(i) / float64(vertexCount) * (2 * math.Pi))),
			float32(math.Sin(float64(i) / float64(vertexCount) * (2 * math.Pi))),
			0,
		}

		for j := range jointsPerVertex {
			jointWeights.Value()[i*jointsPerVertex+j] = Uhh{
				Index:  uint32(j),
				Weight: 0.5, // / float32(jointsPerVertex),
			}
		}
	}

	pose.Value()[0] = gmath.TRS3{
		T: gmath.Vec3{0, 0, 1},
		R: gmath.Rot3One(),
		S: gmath.Vec3Ones(),
	}.Mat4x4()
	// pose.Value()[1] = gmath.TRS3{
	// 	T: gmath.Vec3{0, 0, 0},
	// 	R: gmath.Rot3One(),
	// 	S: gmath.Vec3Ones(),
	// }.Mat4x4()

	jq := new(gpu.JobQueue)
	EnqueueSkinMesh(jq, skinnedPositions, restPositions, jointWeights, jointsPerVertex, pose)
	gpu.WaitForIdle(jq)

	log.Println(skinnedPositions.Value())
}
