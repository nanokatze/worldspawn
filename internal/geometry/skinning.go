// TODO: rename this package to something else to make it clearer that it
// concerns itself with processing geometry. E.g. geonodes?
package geometry

import (
	"os"
	"structs"
	"sync"

	"worldspawn/gpu"
	"worldspawn/internal/gmath"
)

type Uhh struct {
	_      structs.HostLayout
	Index  uint32
	Weight float32
}

type skinMeshEnv struct {
	_ structs.HostLayout

	SkinnedPositions gpu.Pointer[[3]float32]

	RestPositions gpu.Pointer[[3]float32]

	JointWeights    gpu.Pointer[Uhh]
	JointsPerVertex uint32

	VertexCount uint32

	Pose gpu.Pointer[gmath.Mat4x4]
}

var skinMesh = sync.OnceValue(func() *gpu.ComputeShader[skinMeshEnv] {
	return gpu.CompileComputeShader[skinMeshEnv](mustReadFile("/home/nanokatze/code/worldspawn/shaders/geometry_skinning.spv"), "skinMesh")
})

func EnqueueSkinMesh(jq *gpu.JobQueue, skinned, rest gpu.Slice[[3]float32], jointWeights gpu.Slice[Uhh], jointsPerVertex int, pose gpu.Slice[gmath.Mat4x4]) {
	n := gpu.SliceLen(skinned)
	if gpu.SliceLen(rest) != n {
		panic("bad")
	}

	gpu.ParallelFor(jq, []int{(n + 64 - 1) / 64},
		skinMesh().Bind(skinMeshEnv{
			SkinnedPositions: gpu.SliceData(skinned),
			RestPositions:    gpu.SliceData(rest),

			JointWeights:    gpu.SliceData(jointWeights),
			JointsPerVertex: uint32(jointsPerVertex),

			VertexCount: uint32(n),

			Pose: gpu.SliceData(pose),
		}))
}

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}
