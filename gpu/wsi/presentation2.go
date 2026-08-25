package wsi

import (
	"fmt"
	"math"
	"sync"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

var present2Mu sync.Mutex

// TODO: experiment with presenting images directly. This will always require a
// copy on current Vulkan, but will let us sketch out the API. Overall the API
// should be kinda comparable to wl_surface_attach, i.e. quite minimal.
func (swapchain *Swapchain) Present2(jq *gpu.JobQueue, image *gpu.Image) bool {
	present2Mu.Lock()
	defer present2Mu.Unlock()

	// TODO: re-create the swapchain here, if the format, extent, etc differ.

	fence := swapchain.acquireFence

	var index uint32
	if err := gpu.VkFns.AcquireNextImage2KHR(gpu.Device, &vk.AcquireNextImageInfoKHR{
		SType:      vk.STRUCTURE_TYPE_ACQUIRE_NEXT_IMAGE_INFO_KHR,
		Swapchain:  swapchain.vkSwapchain,
		Timeout:    math.MaxUint64,
		Fence:      fence,
		DeviceMask: 0b1,
	}, &index); err != nil {
		panic(fmt.Sprintf("gpu: vkAcquireNextImage2KHR: %v", err))
	}

	if err := gpu.VkFns.WaitForFences(gpu.Device, 1, &fence, vk.TRUE, math.MaxUint64); err != nil {
		panic(fmt.Sprintf("gpu: vkWaitForFences: %v", err))
	}
	if err := gpu.VkFns.ResetFences(gpu.Device, 1, &fence); err != nil {
		panic(fmt.Sprintf("gpu: vkResetFences: %v", err))
	}

	swapchain.images[index].EnqueueInit(jq)
	gpu.EnqueueCopyImage(jq,
		swapchain.images[index], nil,
		image, nil,
		image.Extent())
	swapchain.images[index].EnqueueTransitionLayout(jq, vk.IMAGE_LAYOUT_GENERAL, vk.IMAGE_LAYOUT_PRESENT_SRC_KHR)

	// Establish sync between work and presentation.
	//
	// TODO: do this correctly. The current approach isn't correct and might
	// deadlock the thread, we need to establish sync with e.g. vkQueuePresent.
	gpu.WaitForIdle(jq)

	// TODO: properly sync this so all vkQueuePresents to this swapchain are in
	// order
	jq.Enqueue(&presentJob{
		swapchain: swapchain,
		index:     index,
	})

	return true
}
