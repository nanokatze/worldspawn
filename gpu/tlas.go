package gpu

import (
	"structs"

	"worldspawn/gpu/vk"
)

// TODO: rename to just Instance when we move it into raytracing subpackage
type BLASInstance struct {
	_                 structs.HostLayout
	Transform         [3][4]float32
	InstanceIDAndMask uint32 // TODO: make packing this more user-friendly.
	SBTOffsetAndFlags uint32
	accel             UnsafePointer
}

// TODO: introduce type BLASData struct { p UnsafePointer } or similar and
// remove this method?
func (instance *BLASInstance) SetAccel(blas BLAS) {
	instance.accel = blas.data
}

type TLAS struct {
	tlas struct{}
	accel
}

func NewTLAS(size int) TLAS {
	return TLAS{
		accel: newAccel(size),
	}
}

func (tlas *TLAS) EnqueueBuild(jq *JobQueue, config *AccelBuildConfig) {
	sizes := config.CalcSizes()
	if sizes.Accel > tlas.Size() {
		panic("bad")
	}

	scratch := UnsafePointer(SliceData(MakeSliceUncached[byte](sizes.BuildScratch)))
	defer jq.Cleanup(func() { Free(scratch) })

	vkGeometries := make([]vk.AccelerationStructureGeometryKHR, len(config.Inputs))
	vkBuildRanges := make([]vk.AccelerationStructureBuildRangeInfoKHR, len(config.Inputs))
	for i, input := range config.Inputs {
		input.vkAccelerationStructureGeometry(&vkGeometries[i], &vkBuildRanges[i].PrimitiveCount)
	}

	jq.Enqueue(&accelBuildJob{
		dst:           tlas.accel,
		asType:        config.Type,
		vkGeometries:  vkGeometries,
		vkBuildRanges: vkBuildRanges,
		scratch:       scratch,
	})
}
