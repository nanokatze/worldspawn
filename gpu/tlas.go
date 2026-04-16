package gpu

import (
	"structs"
)

// TODO: rename to just Instance when we move it into raytracing subpackage
type AccelInstance struct {
	_                 structs.HostLayout
	Transform         [3][4]float32
	InstanceIDAndMask uint32 // TODO: make packing this more user-friendly.
	SBTOffsetAndFlags uint32
	accel             UnsafePointer
}

// TODO: or make a With() thing instead? Or introduce BLASPointer or whatever.
func (instance *AccelInstance) SetAccel(blas BLAS) {
	instance.accel = blas.data
}

// TODO: hide the internals i.e. make it be struct{ tlas TLAS }
type TLAS Accel

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
