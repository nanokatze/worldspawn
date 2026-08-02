package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type copyJob struct {
	dst UnsafePointer
	src UnsafePointer
	len int
}

/*
func EnqueueCopy[T any](jq *JobQueue, dst, src Pointer[T]) {
	jq.Enqueue(&copyJob{
		Dst: UnsafePointer(dst),
		Src: UnsafePointer(src),
		Len: int(unsafe.Sizeof(*new(T))),
	})
}
*/

// TODO: rename to EnqueueCopySlice? Alternatively with some hackery we could
// make this work both for slices and pointers.
func EnqueueCopy[T any](jq *JobQueue, dst, src Slice[T]) {
	if SliceLen(dst) == 0 || SliceLen(src) == 0 {
		return
	}

	// TODO: implement memmove semantics or at least overlap checking with panic
	jq.Enqueue(&copyJob{
		dst: UnsafePointer(SliceData(dst)),
		src: UnsafePointer(SliceData(src)),
		len: min(SliceLen(dst), SliceLen(src)) * int(unsafe.Sizeof(*new(T))),
	})
}

func (*copyJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: topology.QueueFamilies(0b100),
	}
}

func (job *copyJob) Exec(q *DeviceQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		// We don't use sparse for now so we can use FULLY_BOUND. In the future
		// when we get sparse going we'll want to set FULLY_BOUND flag by
		// consulting the allocator. RADV benefits from this flag in some cases.

		region := &vk.DeviceMemoryCopyKHR{
			SType: vk.STRUCTURE_TYPE_DEVICE_MEMORY_COPY_KHR,
			SrcRange: vk.DeviceAddressRangeKHR{
				Address: vk.DeviceAddress(job.src),
				Size:    vk.DeviceSize(job.len),
			},
			SrcFlags: vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_FULLY_BOUND_BIT_KHR) |
				vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_UNKNOWN_STORAGE_BUFFER_USAGE_BIT_KHR),
			DstRange: vk.DeviceAddressRangeKHR{
				Address: vk.DeviceAddress(job.dst),
				Size:    vk.DeviceSize(job.len),
			},
			DstFlags: vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_FULLY_BOUND_BIT_KHR) |
				vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_UNKNOWN_STORAGE_BUFFER_USAGE_BIT_KHR),
		}
		pinner.Pin(region)

		vkFns.CmdCopyMemoryKHR(cb, &vk.CopyDeviceMemoryInfoKHR{
			SType:       vk.STRUCTURE_TYPE_COPY_BUFFER_INFO_2,
			RegionCount: 1,
			PRegions:    region,
		})
	})
}
