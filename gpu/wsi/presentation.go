package gpuwsi // TODO: give this package a nicer name

import (
	"fmt"
	"iter"
	"log"
	"math"
	"math/bits"
	"runtime"
	"slices"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/internal/sdl"
)

type Swapchain struct {
	vkSurface     vk.SurfaceKHR
	queueFamilies uint32
	vkSwapchain   vk.SwapchainKHR
	images        []*gpu.Image

	// TODO: remove this stuff from here and probably just have a pool of these things
	acquireFence vk.Fence
}

type SwapchainConfig struct {
	Window       *sdl.Window
	ColorSpace   vk.ColorSpaceKHR
	Format       vk.Format
	Extent       [2]int
	ImageOptions []gpu.ImageOption
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

var device vk.Device
var vkFns struct {
	vk.InstanceFuncs
	vk.DeviceFuncs
}
var initOnce sync.Once

func gpuInit() {
	initOnce.Do(func() {
		device = gpu.Device()
		vkFns.DeviceFuncs.Init(device)
	})
}

func newSwapchain(window *sdl.Window) *Swapchain {
	gpuInit()

	vkSurface, err := sdlVulkanCreateSurface(window, gpu.Instance(), nil)
	if err != nil {
		// TODO: we might want to handle this, actually.
		panic(fmt.Sprintf("gpu: SDL_Vulkan_CreateSurface: %v", err))
	}

	var mask uint32
	for family := range ones32(gpu.QueueFamilies(0)) {
		var supported vk.Bool32
		if err := vkFns.GetPhysicalDeviceSurfaceSupportKHR(gpu.PhysicalDevice(), uint32(family), vkSurface, &supported); err != nil {
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

func ones32(x uint32) iter.Seq[int] {
	return func(yield func(int) bool) {
		i := 0
		for {
			i += bits.TrailingZeros32(x >> i)
			if i >= 32 {
				break
			}
			if !yield(i) {
				return
			}
			i++
		}
	}
}

// TODO: add a method to get current swapchain config like e.g. format, extent
// and everything?

// TODO: just mirror vk api?
// TODO: redo how swapchain constructor api looks like
func (swapchain *Swapchain) reconfigure(config *SwapchainConfig) *Swapchain {
	// TODO: properly choose minImageCount, or let the user do it
	minImageCount := uint32(4)

	imgConf := gpu.JoinImageOptions(config.Format, config.Extent[:], config.ImageOptions...)

	allQueueFamilies := slices.Collect(func(yield func(uint32) bool) {
		for family := range ones32(gpu.QueueFamilies(0)) {
			yield(uint32(family))
		}
	})

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
		ImageExtent:           vk.Extent2D{Width: uint32(config.Extent[0]), Height: uint32(config.Extent[1])},
		ImageArrayLayers:      uint32(imgConf.Layers),
		ImageUsage:            imgConf.Usages,
		ImageSharingMode:      vk.SHARING_MODE_CONCURRENT,
		QueueFamilyIndexCount: uint32(len(allQueueFamilies)),
		PQueueFamilyIndices:   pinnedSliceData(&pinner, allQueueFamilies),
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

	images := make([]*gpu.Image, len(vkImages))
	for i, vkImage := range vkImages {
		images[i] = gpu.NewImageFromVkImage(vkImage, config.Format, config.Extent[:], config.ImageOptions...)
	}

	return &Swapchain{
		vkSurface:     swapchain.vkSurface,
		queueFamilies: swapchain.queueFamilies,
		vkSwapchain:   vkSwapchain,
		images:        images,

		acquireFence: swapchain.acquireFence,
	}
}

func (swapchain *Swapchain) Image(index int) *gpu.Image {
	return swapchain.images[index]
}

// TODO: rename to AcquireNextImage
// TODO: use int here, and or return error
// TODO: should take gpu.WaitGroup to signal
func (swapchain *Swapchain) Acquire() int {
	var index uint32
	if err := vkFns.AcquireNextImage2KHR(device, &vk.AcquireNextImageInfoKHR{
		SType:      vk.STRUCTURE_TYPE_ACQUIRE_NEXT_IMAGE_INFO_KHR,
		Swapchain:  swapchain.vkSwapchain,
		Timeout:    math.MaxUint64,
		Fence:      swapchain.acquireFence,
		DeviceMask: 0b1,
	}, &index); err != nil && err != vk.SUBOPTIMAL_KHR {
		if err != vk.ERROR_OUT_OF_DATE_KHR {
			panic(fmt.Sprintf("gpu: vkAcquireNextImage2KHR: %v", err))
		}
		return -1
	}

	// TODO: get rid of this
	fence := swapchain.acquireFence
	if err := vkFns.WaitForFences(device, 1, &fence, vk.TRUE, math.MaxUint64); err != nil {
		panic(fmt.Sprintf("gpu: vkWaitForFences: %v", err))
	}
	if err := vkFns.ResetFences(device, 1, &fence); err != nil {
		panic(fmt.Sprintf("gpu: vkResetFences: %v", err))
	}

	return int(index)
}

type presentJob struct {
	swapchain *Swapchain
	index     uint32
}

func (swapchain *Swapchain) Present(jq *gpu.JobQueue, index int) {
	swapchain.images[index].EnqueueTransitionLayout(jq, vk.IMAGE_LAYOUT_GENERAL, vk.IMAGE_LAYOUT_PRESENT_SRC_KHR)

	jq.Enqueue(&presentJob{
		swapchain: swapchain,
		index:     uint32(index),
	})

	gpu.WaitForIdle(jq)
}

func (job *presentJob) Info() gpu.JobInfo {
	return gpu.JobInfo{
		QueueFamilies: job.swapchain.queueFamilies,
	}
}

func (job *presentJob) Exec(q *gpu.CommandQueue) {
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

// TODO: remove pinned* stuff
func pinned[T any](pinner *runtime.Pinner, p *T) *T {
	pinner.Pin(p)
	return p
}

func pinnedSliceData[T any](pinner *runtime.Pinner, s []T) *T {
	return pinned(pinner, unsafe.SliceData(s))
}

// TODO: rename to something better
// TODO: kill
func enumerate[T any](f func(len *uint32, data *T) error) ([]T, error) {
	return enumerate2(nil, f)
}

// TODO: rename to something better
func enumerate2[T any](init func([]T), f func(len *uint32, data *T) error) ([]T, error) {
	var len uint32
	if err := f(&len, nil); err != nil {
		return nil, err
	}
	data := make([]T, int(len))
	if init != nil {
		init(data)
	}
	if err := f(&len, unsafe.SliceData(data)); err != nil {
		return nil, err
	}
	return data[:len], nil
}
