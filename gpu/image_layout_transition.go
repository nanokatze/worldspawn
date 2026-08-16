package gpu

import (
	"runtime"
	"sync"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: also include QFOT capabilities for transfering to/from EXTERNAL/FOREIGN
type transitionImageLayoutJob struct {
	imageData   *imageData
	imageBounds imageBounds
	oldLayout   vk.ImageLayout
	newLayout   vk.ImageLayout
}

func (img *Image) EnqueueTransitionLayout(jq *JobQueue, oldLayout, newLayout vk.ImageLayout) {
	jq.Enqueue(&transitionImageLayoutJob{
		imageData:   img.data,
		imageBounds: img.bounds,
		oldLayout:   oldLayout,
		newLayout:   newLayout,
	})
}

// RADV does not implement transition to PRESENT_SRC on queues that don't
// support compute. This is a driver bug.
var transitionToPresentSrcOnTransferQueueIsBroken = sync.OnceValue(func() bool {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	coreProps_1_2 := vk.PhysicalDeviceVulkan12Properties{
		SType: vk.STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_2_PROPERTIES,
	}
	vkFns.GetPhysicalDeviceProperties2(physicalDevice,
		&vk.PhysicalDeviceProperties2{
			SType: vk.STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2,
			PNext: unsafe.Pointer(pinned(&pinner, &coreProps_1_2)),
		})

	return coreProps_1_2.DriverID == vk.DRIVER_ID_MESA_RADV
})

func (job *transitionImageLayoutJob) Info() JobInfo {
	// VUID-vkCmdPipelineBarrier2-commandBuffer-cmdpool
	// The VkCommandPool that commandBuffer was allocated from must support
	// transfer, graphics, compute, decode, or encode operations
	families := topology.QueueFamilies(0b100)
	if job.newLayout == vk.IMAGE_LAYOUT_PRESENT_SRC_KHR && transitionToPresentSrcOnTransferQueueIsBroken() {
		families = topology.QueueFamilies(0b010)
	}
	return JobInfo{QueueFamilies: families}
}

// TODO: group these jobs so we can poke vkCmdPipelineBarrier2 less. On RADV
// there's device overheads arising from our current usage pattern, on the
// transfer-only queue.
func (job *transitionImageLayoutJob) Exec(q *DeviceQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		imageMemoryBarrier := &vk.ImageMemoryBarrier2{
			SType:            vk.STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER_2,
			OldLayout:        job.oldLayout,
			NewLayout:        job.newLayout,
			Image:            job.imageData.vkImage,
			SubresourceRange: job.imageBounds.VkImageSubresourceRange(job.imageData.format),
		}
		pinner.Pin(imageMemoryBarrier)

		vkFns.CmdPipelineBarrier2(cb,
			&vk.DependencyInfo{
				SType:                   vk.STRUCTURE_TYPE_DEPENDENCY_INFO,
				ImageMemoryBarrierCount: 1,
				PImageMemoryBarriers:    imageMemoryBarrier,
			})
	})
}
