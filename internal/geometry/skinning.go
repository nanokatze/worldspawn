// TODO: rename this package to something else to make it clearer that it
// concerns itself with processing geometry. E.g. geonodes?
package geometry

import (
	_ "embed"
	"structs"
	"sync"

	"worldspawn/gpu"
	"worldspawn/internal/gmath"
)

//go:generate slangc -target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -fvk-use-c-layout -matrix-layout-row-major -capability vk_mem_model -capability spvDescriptorHeapEXT -o _shaders.spv skinning.slang

//go:embed _shaders.spv
var _shaders []byte

type Uhh struct {
	_      structs.HostLayout
	Index  uint32
	Weight float32
}

type skinMeshEnv struct {
	_ structs.HostLayout

	SkinnedPositions gpu.Pointer[[3]float32]
	SkinnedNormals   gpu.Pointer[[3]float32]

	RestPositions gpu.Pointer[[3]float32]
	RestNormals   gpu.Pointer[[3]float32]

	JointWeights    gpu.Pointer[Uhh]
	JointsPerVertex uint32

	VertexCount uint32

	Pose gpu.Pointer[gmath.Mat4x4f32]
}

var skinMesh = sync.OnceValue(func() *gpu.ComputeShader[skinMeshEnv] {
	return gpu.CompileComputeShader[skinMeshEnv](_shaders, "skinMesh")
})

func EnqueueSkinMesh(
	jq *gpu.JobQueue,
	skinnedPositions gpu.Slice[[3]float32],
	skinnedNormals gpu.Slice[[3]float32],
	restPositions gpu.Slice[[3]float32],
	restNormals gpu.Slice[[3]float32],
	jointWeights gpu.Slice[Uhh],
	jointsPerVertex int,
	pose gpu.Slice[gmath.Mat4x4f32]) {
	n := gpu.SliceLen(skinnedPositions)
	if gpu.SliceLen(skinnedNormals) != n ||
		gpu.SliceLen(restPositions) != n || gpu.SliceLen(restNormals) != n {
		panic("bad")
	}

	gpu.ParallelFor(jq, []int{(n + 64 - 1) / 64},
		skinMesh().Bind(skinMeshEnv{
			SkinnedPositions: gpu.SliceData(skinnedPositions),
			SkinnedNormals:   gpu.SliceData(skinnedNormals),

			RestPositions: gpu.SliceData(restPositions),
			RestNormals:   gpu.SliceData(restNormals),

			JointWeights:    gpu.SliceData(jointWeights),
			JointsPerVertex: uint32(jointsPerVertex),

			VertexCount: uint32(n),

			Pose: gpu.SliceData(pose),
		}))
}
