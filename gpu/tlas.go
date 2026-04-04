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

// WithTransform(stuff), WithInstanceID(instanceID uint32), WithMask(mask uint8)
// func MakeTLASInstance(opts ...TLASOption)

// TODO: or make a constructor instead? Func vararg constructor would've been
// pretty nice I guess.
func (instance *AccelInstance) SetAccel(accel Accel) {
	instance.accel = accel.data
}

// TODO: accel build config. We'll need to split config into two parts, one used
// to calculate the size and the other would actually equip it with data. This
// is necessary so that the user can allocate a random buffer for their own use
// if they need to. Or we could have TLASConfig and TLASBuildConfig types, but
// idk.

/*
type TLAS struct{ tlas Accel }

type TLASBuildConfig struct {
	// TODO: allow this to either be Slice[AccelInstance] or Slice[Pointer[AccelInstance]]
	Instances Slice[AccelInstance]
}

func (buildConfig TLASBuildConfig) EnqueueBuild(jq *JobQueue, out TLAS) {
}
*/
