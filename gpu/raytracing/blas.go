package raytracing

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type BLAS struct {
	blas struct{}
	accel
}

func NewBLAS(size int) BLAS {
	return BLAS{
		accel: newAccel(size),
	}
}

// TODO: accept scratch explicitly as a byte slice or something similar
func (blas *BLAS) EnqueueBuild(jq *gpu.JobQueue, config *AccelBuildConfig) {
	sizes := config.CalcSizes()
	if sizes.Accel > blas.Size() {
		panic("bad")
	}

	scratch := gpu.UnsafePointer(gpu.SliceData(gpu.MakeSliceUncached[byte](sizes.BuildScratch)))
	defer jq.Cleanup(func() { gpu.Free(scratch) })

	vkGeometries := make([]vk.AccelerationStructureGeometryKHR, len(config.Inputs))
	vkBuildRanges := make([]vk.AccelerationStructureBuildRangeInfoKHR, len(config.Inputs))
	for i, input := range config.Inputs {
		input.vkAccelerationStructureGeometry(&vkGeometries[i], &vkBuildRanges[i].PrimitiveCount)
	}

	jq.Enqueue(&accelBuildJob{
		dst:           blas.accel,
		asType:        config.Type,
		vkGeometries:  vkGeometries,
		vkBuildRanges: vkBuildRanges,
		scratch:       scratch,
	})
}
