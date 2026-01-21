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

		dstBuffer, dstOffset := BufferAndOffset(job.dst)
		srcBuffer, srcOffset := BufferAndOffset(job.src)

		region := &vk.BufferCopy2{
			SType:     vk.STRUCTURE_TYPE_BUFFER_COPY_2,
			SrcOffset: srcOffset,
			DstOffset: dstOffset,
			Size:      vk.DeviceSize(job.len),
		}
		pinner.Pin(region)

		vkFns.CmdCopyBuffer2(cb, &vk.CopyBufferInfo2{
			SType:       vk.STRUCTURE_TYPE_COPY_BUFFER_INFO_2,
			SrcBuffer:   srcBuffer,
			DstBuffer:   dstBuffer,
			RegionCount: 1,
			PRegions:    region,
		})
	})
}
