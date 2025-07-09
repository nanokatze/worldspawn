package gpu

import (
	"fmt"
	"log"
	"math"
	"runtime"

	"worldspawn/gpu/vk"
	"worldspawn/sdl"
	sdl_vulkan "worldspawn/sdl/vulkan"
)

// TODO: experiment with and propose "displayable image" extension to VK WG.

// TODO: move this garbage to its own sdl_swapchain containment package. We
// could perhaps also get rid of sdl_vulkan that way.

type Swapchain struct {
	vkSurface     vk.SurfaceKHR
	queueFamilies uint32
	vkSwapchain   vk.SwapchainKHR
	images        []*Image

	// TODO: remove this stuff from here and probably just have a pool of these things
	acquireFence vk.Fence
}

type SwapchainConfig struct {
	Window     *sdl.Window
	ColorSpace vk.ColorSpaceKHR
	*ImageConfig
	OldSwapchain *Swapchain
}

// TODO: isn't passing both the window and oldSwapchain a bit redundant? Yeah,
// it would be nice if we could avoid passing oldSwapchain, but...
func NewSwapchain(config *SwapchainConfig) *Swapchain {
	oldSwapchain := config.OldSwapchain
	if oldSwapchain == nil {
		oldSwapchain = newSwapchain(config.Window)
	}
	return oldSwapchain.reconfigure(config)
}

func newSwapchain(window *sdl.Window) *Swapchain {
	gpuInit()

	vkSurface, err := sdl_vulkan.CreateSurface(window, sdl_vulkan.Instance(instance), nil)
	if err != nil {
		// TODO: we might want to handle this, actually.
		panic(fmt.Sprintf("gpu: SDL_Vulkan_CreateSurface: %v", err))
	}

	var mask uint32
	// TODO: stop using the deprecated All()
	for _, family := range queueFamilies.All() {
		var supported vk.Bool32
		if err := vkFns.GetPhysicalDeviceSurfaceSupportKHR(physicalDevice, family, vk.SurfaceKHR(vkSurface), &supported); err != nil {
			panic(fmt.Sprintf("gpu: vkGetPhysicalDeviceSurfaceSupportKHR: %v", err))
		}
		// TODO: make a func for converting from Bool32
		if supported != vk.FALSE {
			mask |= 1 << family
		}
	}

	// TODO: print this as a sequence of integers rather than a bitmask
	log.Printf("can present from families 0b%b", mask)

	var acquireFence vk.Fence
	if err := vkFns.CreateFence(device, &vk.FenceCreateInfo{
		SType: vk.STRUCTURE_TYPE_FENCE_CREATE_INFO,
	}, nil, &acquireFence); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateFence: %v", err))
	}

	return &Swapchain{
		vkSurface:     vk.SurfaceKHR(vkSurface),
		queueFamilies: mask,

		acquireFence: acquireFence,
	}
}

// TODO: add a method to get current swapchain config like e.g. format, extent
// and everything?

// TODO: just mirror vk api?
// TODO: redo how swapchain constructor api looks like
func (swapchain *Swapchain) reconfigure(config *SwapchainConfig) *Swapchain {
	// TODO: properly choose minImageCount, or let the user do it
	minImageCount := uint32(4)

	var pinner runtime.Pinner
	defer pinner.Unpin()

	var vkSwapchain vk.SwapchainKHR
	if err := vkFns.CreateSwapchainKHR(device, &vk.SwapchainCreateInfoKHR{
		SType: vk.STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
		// Flags:                 vk.SwapchainCreateFlagsKHR(vk.SWAPCHAIN_CREATE_DEFERRED_MEMORY_ALLOCATION_BIT_EXT),
		Surface:               swapchain.vkSurface,
		MinImageCount:         minImageCount,
		ImageFormat:           config.Format,
		ImageColorSpace:       config.ColorSpace,
		ImageExtent:           vk.Extent2D{Width: uint32(config.Extent.X), Height: uint32(config.Extent.Y)},
		ImageArrayLayers:      uint32(config.Layers),
		ImageUsage:            config.Usage.vkImageUsageFlags(config.Format),
		ImageSharingMode:      vk.SHARING_MODE_CONCURRENT,
		QueueFamilyIndexCount: uint32(len(queueFamilies.All())),
		PQueueFamilyIndices:   pinnedSliceData(&pinner, queueFamilies.All()),
		PreTransform:          vk.SURFACE_TRANSFORM_IDENTITY_BIT_KHR,
		CompositeAlpha:        vk.COMPOSITE_ALPHA_OPAQUE_BIT_KHR,
		PresentMode:           vk.PRESENT_MODE_FIFO_KHR,
		Clipped:               vk.TRUE,
		OldSwapchain:          swapchain.vkSwapchain,
	}, nil, &vkSwapchain); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateSwapchainKHR: %v", err))
	}

	vkImages, err := enumerate(func(len *uint32, data *vk.Image) error {
		return vkFns.GetSwapchainImagesKHR(device, vkSwapchain, len, data)
	})
	if err != nil {
		panic(fmt.Sprintf("gpu: vkGetSwapchainImagesKHR: %v", err))
	}

	images := make([]*Image, len(vkImages))
	for i, vkImage := range vkImages {
		// TODO: we should introduce an Image constructor for imported/foreign
		// images.
		imageData := new(imageData)
		imageData.vkImage = vkImage
		imageData.dim = ImageDim2D
		imageData.extent = config.Extent
		imageData.layers = uint32(config.Layers)
		imageData.mipLevels = 1
		imageData.format = config.Format
		imageData.usage = config.Usage
		images[i] = newImage(
			imageData,
			imageData.dim,
			imageData.format,
			0, int(imageData.mipLevels),
			0, int(imageData.layers))
	}

	return &Swapchain{
		vkSurface:     swapchain.vkSurface,
		queueFamilies: swapchain.queueFamilies,
		vkSwapchain:   vkSwapchain,
		images:        images,

		acquireFence: swapchain.acquireFence,
	}
}

func (swapchain *Swapchain) Image(index uint32) *Image {
	return swapchain.images[index]
}

// TODO: rename to AcquireNextImage
// TODO: use int here, and or return error
// TODO: should take gpu.WaitGroup to signal
func (swapchain *Swapchain) Acquire() uint32 {
	var index uint32
	if err := vkFns.AcquireNextImage2KHR(device, &vk.AcquireNextImageInfoKHR{
		SType:      vk.STRUCTURE_TYPE_ACQUIRE_NEXT_IMAGE_INFO_KHR,
		Swapchain:  swapchain.vkSwapchain,
		Timeout:    math.MaxUint64,
		Fence:      swapchain.acquireFence,
		DeviceMask: 0b1,
	}, &index); err != nil {
		panic(fmt.Sprintf("gpu: vkAcquireNextImage2KHR: %v", err))
	}

	// TODO: get rid of this
	fence := swapchain.acquireFence
	if err := vkFns.WaitForFences(device, 1, &fence, vk.TRUE, math.MaxUint64); err != nil {
		panic(fmt.Sprintf("gpu: vkWaitForFences: %v", err))
	}
	if err := vkFns.ResetFences(device, 1, &fence); err != nil {
		panic(fmt.Sprintf("gpu: vkResetFences: %v", err))
	}

	return index
}

type presentJob struct {
	swapchain *Swapchain
	index     uint32
}

/*
func (swapchain *Swapchain) Present(cq *JobQueue, index uint32, presentMode vk.PresentModeKHR) (ok bool) {
	WakeyWakeyUrgent()

	var pinner runtime.Pinner
	defer pinner.Unpin()

	// TODO: missing sync
	if err := vkFns.QueuePresentKHR(gfx.vkQueue, &vk.PresentInfoKHR{
		SType:          vk.STRUCTURE_TYPE_PRESENT_INFO_KHR,
		SwapchainCount: 1,
		PSwapchains:    pinned(&pinner, &swapchain.vkSwapchain),
		PImageIndices:  pinned(&pinner, &index),
	}); err != nil {
		panic(fmt.Sprintf("gpu: vkQueuePresentKHR: %v", err))
	}

	return true
}
*/

func (job *presentJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.swapchain.queueFamilies,
	}
}

func (job *presentJob) Exec(q *CommandQueue) {
	// log.Print("execing present job on queue family ", q.queueFamily)
	q.QueueOperation(func(vkQueue vk.Queue) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		// TODO: we need to insert on queue signal correctly
		if err := vkFns.QueuePresentKHR(vkQueue, &vk.PresentInfoKHR{
			SType:          vk.STRUCTURE_TYPE_PRESENT_INFO_KHR,
			SwapchainCount: 1,
			PSwapchains:    pinned(&pinner, &job.swapchain.vkSwapchain),
			PImageIndices:  pinned(&pinner, &job.index),
		}); err != nil {
			panic(fmt.Sprintf("gpu: vkQueuePresentKHR: %v", err))
		}
	})
}
