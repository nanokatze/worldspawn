package gpu

import (
	"structs"

	"worldspawn/gpu/vk"
)

// TODO: rename to just Instance when we move it into raytracing subpackage
type AccelInstance struct {
	_                 structs.HostLayout
	Transform         [3][4]float32
	InstanceIDAndMask uint32 // TODO: make packing this more user-friendly.
	SBTOffsetAndFlags uint32
	accel             UnsafePointer
}

// TODO: introduce type AccelPtr struct { p UnsafePointer } or similar for the
// accel field, and remove this method?
func (instance *AccelInstance) SetAccel(blas BLAS) {
	instance.accel = blas.data
}

type TLAS struct {
	tlas struct{}
	Accel
}

// TODO: rename this to make it clear that this works in terms of instances rather than bytes.
func NewTLAS(maxInstances int) TLAS {
	config := &AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
		Inputs: []AccelBuildInput{
			&AccelBuildInputInstances{
				InstanceCount: uint32(maxInstances),
			},
		},
	}
	return TLAS{
		Accel: NewAccel(config.CalcSizes().Accel),
	}
}

func (tlas *TLAS) EnqueueBuild(jq *JobQueue, config *AccelBuildConfig) {
	sizes := config.CalcSizes()
	if sizes.Accel > tlas.Size() {
		panic("bad")
	}
	scratch := UnsafePointer(SliceData(MakeSliceUncached[byte](sizes.BuildScratch)))
	defer jq.Cleanup(func() { Free(scratch) })
	EnqueueAccelBuild(jq, tlas.data, tlas.size, config, scratch)
}

// TODO: accel build config. We'll need to split config into two parts, one used
// to calculate the size and the other would actually equip it with data. This
// is necessary so that the user can allocate a random buffer for their own use
// if they need to. Or we could have TLASConfig and TLASBuildConfig types, but
// idk.

/*
type TLASBuildConfig struct {
	// TODO: allow this to either be Slice[AccelInstance] or Slice[Pointer[AccelInstance]]
	Instances Slice[AccelInstance]
}

func (buildConfig TLASBuildConfig) EnqueueBuild(jq *JobQueue, out TLAS) {
}
*/
