package gpu

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: move all image stuff into a subpackage

type ImageDim int // uint8?

const (
	_ ImageDim = iota
	ImageDim1D
	ImageDim2D
	ImageDim3D
	ImageDimCube // a special kind of 2D image
)

func (dim ImageDim) vkImageType() vk.ImageType {
	switch dim {
	case ImageDim1D:
		return vk.IMAGE_TYPE_1D
	case ImageDim2D, ImageDimCube:
		return vk.IMAGE_TYPE_2D
	case ImageDim3D:
		return vk.IMAGE_TYPE_3D
	default:
		panic("unreachable")
	}
}

func (dim ImageDim) vkImageViewType() vk.ImageViewType {
	switch dim {
	case ImageDim1D:
		return vk.IMAGE_VIEW_TYPE_1D_ARRAY
	case ImageDim2D:
		return vk.IMAGE_VIEW_TYPE_2D_ARRAY
	case ImageDimCube:
		return vk.IMAGE_VIEW_TYPE_CUBE_ARRAY
	case ImageDim3D:
		return vk.IMAGE_VIEW_TYPE_3D
	default:
		panic("unreachable")
	}
}

type ImageUsage uint8

const (
	_ ImageUsage = 1 << iota >> 1

	// Not specifying this usage flag on attachments allows for better
	// performance on some implementations, such as Intel's.
	//
	// TODO: provide source
	ImageUsageSampling

	ImageUsageLoadStore

	ImageUsageAttachment
)

func (usage ImageUsage) vkImageUsageFlags(format Format) vk.ImageUsageFlags {
	vkUsage := vk.ImageUsageFlags(0)

	vkUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_DST_BIT) | vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_SRC_BIT)

	if usage&ImageUsageSampling != 0 {
		vkUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT)
	}

	if usage&ImageUsageLoadStore != 0 {
		vkUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT)
	}

	if usage&ImageUsageAttachment != 0 {
		aspects := formatutil.Aspects(format)
		if aspects&vk.ImageAspectFlags(vk.IMAGE_ASPECT_COLOR_BIT) != 0 {
			vkUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT)
		}
		if aspects&(vk.ImageAspectFlags(vk.IMAGE_ASPECT_DEPTH_BIT)|vk.ImageAspectFlags(vk.IMAGE_ASPECT_STENCIL_BIT)) != 0 {
			vkUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)
		}
	}

	return vkUsage
}

// TODO: do order layer before mip level throughout our api

// TODO: do we need any other params here?
type ImageConfig struct {
	Dim       ImageDim
	Extent    Int3
	Layers    int
	MipLevels int
	Samples   uint32 // TODO: remove multisampled images altogether from our abstraction?
	Format    Format
	// TODO: rename into something like "AdditionalUsage"?
	Usage ImageUsage // TODO: a separate stencil usage? We could also pack usage for stencil into the high bits
}

func (config *ImageConfig) vkImageCreateInfo(
	queueFamilies []uint32,
	createInfo *vk.ImageCreateInfo) {
	flags := vk.ImageCreateFlags(0)

	flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_MUTABLE_FORMAT_BIT)

	if config.Dim == ImageDim2D &&
		config.Extent.Y == config.Extent.Z &&
		config.Layers >= 6 &&
		config.Samples == 1 {
		flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_CUBE_COMPATIBLE_BIT)
	}

	// TODO: actually check if it's compressed instead of checking BlockExtent
	if formatutil.Describe(config.Format).BlockExtent != (vk.Extent3D{Width: 1, Height: 1, Depth: 1}) {
		flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_BLOCK_TEXEL_VIEW_COMPATIBLE_BIT)
	}

	flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_EXTENDED_USAGE_BIT)

	*createInfo = vk.ImageCreateInfo{
		SType:                 vk.STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		Flags:                 flags,
		ImageType:             config.Dim.vkImageType(),
		Format:                config.Format,
		Extent:                int3ToVkExtent3D(config.Extent),
		MipLevels:             uint32(config.MipLevels),
		ArrayLayers:           uint32(config.Layers),
		Samples:               vk.SampleCountFlagBits(config.Samples),
		Tiling:                vk.IMAGE_TILING_OPTIMAL,
		Usage:                 config.Usage.vkImageUsageFlags(config.Format),
		SharingMode:           vk.SHARING_MODE_CONCURRENT,
		QueueFamilyIndexCount: uint32(len(queueFamilies)),
		PQueueFamilyIndices:   unsafe.SliceData(queueFamilies),
		InitialLayout:         vk.IMAGE_LAYOUT_UNDEFINED,
	}
}

type Image struct {
	base *imageData

	dim          ImageDim
	format       Format // see SubImage
	baseLayer    uint32
	layers       uint32
	baseMipLevel uint8 // TODO: compress these into a single uint64
	mipLevels    uint8

	// precomputed stuff

	extent Int3 // TODO: change to Extent3D for a more compact representation?

	owner bool
}

type imageData struct {
	vkImage vk.Image

	// TODO: review which of these fields we need
	dim       ImageDim
	extent    Int3 // TODO: change this back to vk.Extent3D?
	layers    uint32
	mipLevels uint32
	format    Format
	usage     ImageUsage

	memory *deviceMemory // TODO: replace with an UnsafePointer and length
}

// TODO: lazily allocate these?
var imageDescAllocHint int64
var imageDescriptors = newSlotAlloc(1e6)
var imageViews = make([]uint64, 1e6)

func NewImage(config *ImageConfig) *Image {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	imageData := new(imageData)

	imageCreateInfo := new(vk.ImageCreateInfo)
	config.vkImageCreateInfo(queueFamilies.probe, imageCreateInfo)

	pinner.Pin(imageCreateInfo)
	pinner.Pin(imageCreateInfo.PQueueFamilyIndices)

	requirements := &vk.MemoryRequirements2{
		SType: vk.STRUCTURE_TYPE_MEMORY_REQUIREMENTS_2,
	}
	vkFns.GetDeviceImageMemoryRequirements(device,
		&vk.DeviceImageMemoryRequirements{
			SType:       vk.STRUCTURE_TYPE_DEVICE_IMAGE_MEMORY_REQUIREMENTS,
			PCreateInfo: imageCreateInfo,
		},
		requirements)

	log.Printf("image %v needs %v aligned to %v", config.Extent, requirements.Size, requirements.Alignment)

	size := roundUpDeviceAllocationSize(int(requirements.Size))

	if err := vkFns.CreateImage(device, imageCreateInfo, nil, &imageData.vkImage); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateImage: %v", err))
	}

	memoryTypeIndex := findMemoryTypeIndex(requirements.MemoryTypeBits, 0)

	{
		allocPoolMu.Lock()
		entries := allocPool[size]
		if len(entries) > 0 {
			imageData.memory = entries[len(entries)-1]
			allocPool[size] = entries[:len(entries)-1]
		}
		allocPoolMu.Unlock()
	}
	if imageData.memory == nil {
		var allocation deviceMemory
		allocation.size = size
		if err := vkFns.AllocateMemory(device, &vk.MemoryAllocateInfo{
			SType:           vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
			AllocationSize:  vk.DeviceSize(size),
			MemoryTypeIndex: uint32(memoryTypeIndex),
		}, nil, &allocation.memory); err != nil {
			panic(fmt.Sprintf("gpu: vkAllocateMemory: %v", err))
		}
		imageData.memory = &allocation
	}

	if err := vkFns.BindImageMemory2(device, 1, &vk.BindImageMemoryInfo{
		SType:        vk.STRUCTURE_TYPE_BIND_IMAGE_MEMORY_INFO,
		Image:        imageData.vkImage,
		Memory:       imageData.memory.memory,
		MemoryOffset: 0,
	}); err != nil {
		panic(fmt.Sprintf("gpu: vkBindImageMemory2: %v", err))
	}

	imageData.dim = config.Dim
	imageData.extent = config.Extent
	imageData.layers = uint32(config.Layers)
	imageData.mipLevels = uint32(config.MipLevels)
	imageData.format = config.Format
	imageData.usage = config.Usage

	/*
		runtime.AddCleanup(imageData,
			func(vkImage vk.Image) {

			},
			struct{}{})
	*/

	tmp := newImage(
		imageData,
		imageData.dim,
		imageData.format,
		0, int(imageData.layers),
		0, int(imageData.mipLevels))
	tmp.owner = true
	return tmp
}

func newImage(base *imageData,
	dim ImageDim,
	format Format,
	baseLayer, layers int,
	baseMipLevel, mipLevels int) *Image {
	extent := minify(base.extent, baseMipLevel)
	// TODO: make this less ugly
	block1 := formatutil.Describe(format).BlockExtent
	block2 := formatutil.Describe(base.format).BlockExtent
	if block1 != block2 {
		// Block sizes differ, therefore format classes differ, this can only be
		// possible if we're reinterpreting a compressed format as uncompressed,
		// so block1 must be 1, 1, 1.
		if block1 != (vk.Extent3D{Width: 1, Height: 1, Depth: 1}) {
			panic("bad compressed image reinterpretation")
		}
		extent = divByBlockExtentRoundUp(extent, base.format)
	}

	return &Image{
		base:         base,
		dim:          dim,
		format:       format,
		baseLayer:    uint32(baseLayer),
		layers:       uint32(layers),
		baseMipLevel: uint8(baseMipLevel),
		mipLevels:    uint8(mipLevels),
		extent:       extent,
	}
}

// Format specifies what format to reinterpret this image
//
// TODO: for multi-planar images we'd also want to specify aspect mask. Instead,
// we can pretend all images are multi-planar and specify *plane mask* (up to 3
// bits). Depth-stencil images always have depth be plane 0 and stencil plane 1.
// TODO: better parameter names
func (image *Image) SubImage(
	dim ImageDim,
	format Format,
	baseLayer, endLayer int,
	baseMipLevel, endMipLevel int) *Image {
	// TODO: SubImage-specific validation

	return newImage(
		image.base,
		dim,
		format,
		int(image.baseLayer)+baseLayer, endLayer-baseLayer,
		int(image.baseMipLevel)+baseMipLevel, endMipLevel-baseMipLevel)
}

func (image *Image) Dim() ImageDim { return image.dim }

func (image *Image) Format() Format { return image.format }

func (image *Image) Extent() Int3 { return image.extent }

func (image *Image) EnqueueInit(jq *JobQueue) {
	image.enqueueTransitionLayout(jq, vk.IMAGE_LAYOUT_UNDEFINED, vk.IMAGE_LAYOUT_GENERAL)
}

// TODO: move to its own file I guess. Or idk.
type transitionImageLayoutJob struct {
	imageData        *imageData
	subresourceRange vk.ImageSubresourceRange
	oldLayout        vk.ImageLayout
	newLayout        vk.ImageLayout
}

func (image *Image) enqueueTransitionLayout(jq *JobQueue, oldLayout, newLayout vk.ImageLayout) {
	jq.Enqueue(&transitionImageLayoutJob{
		imageData:        image.base,
		subresourceRange: vkImageSubresourceRange(image),
		oldLayout:        oldLayout,
		newLayout:        newLayout,
	})
}

func (job *transitionImageLayoutJob) Info() JobInfo {
	// VUID-vkCmdPipelineBarrier2-commandBuffer-cmdpool
	// The VkCommandPool that commandBuffer was allocated from must support
	// transfer, graphics, compute, decode, or encode operations
	families := queueFamilies.Mask(0b100)
	if deviceProps.Vulkan12.DriverID == vk.DRIVER_ID_MESA_RADV && job.newLayout == vk.IMAGE_LAYOUT_PRESENT_SRC_KHR {
		// WA: RADV does not implement transition to PRESENT_SRC on queues that
		// don't support compute.
		families = queueFamilies.Mask(0b010)
	}
	return JobInfo{QueueFamilies: families}
}

// TODO: group these jobs so we can poke vkCmdPipelineBarrier2 less. On RADV
// there's device overheads arising from our current usage pattern, on the
// transfer-only queue.
func (job *transitionImageLayoutJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		imageMemoryBarrier := &vk.ImageMemoryBarrier2{
			SType:            vk.STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER_2,
			OldLayout:        job.oldLayout,
			NewLayout:        job.newLayout,
			Image:            job.imageData.vkImage,
			SubresourceRange: job.subresourceRange,
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

func vkImageSubresourceRange(image *Image) vk.ImageSubresourceRange {
	return vk.ImageSubresourceRange{
		AspectMask:     formatutil.Aspects(image.format),
		BaseMipLevel:   uint32(image.baseMipLevel),
		LevelCount:     uint32(image.mipLevels),
		BaseArrayLayer: image.baseLayer,
		LayerCount:     image.layers,
	}
}

// TODO: deprecated in favor of Descriptor()
func (image *Image) NewSamplingView() SamplingView {
	return newSamplingView(image)
}

// TODO: deprecated in favor of Descriptor()
func (image *Image) NewStorageView() StorageView {
	return newStorageView(image)
}

func (image *Image) Destroy() {
	// Ideally we'd not have Destroy(), but eh.
	if image.owner {
		image.base.destroy()
	}
}

func (image *imageData) destroy() {
	vkFns.DestroyImage(device, image.vkImage, nil)

	allocPoolMu.Lock()
	allocPool[image.memory.size] = append(allocPool[image.memory.size], image.memory)
	allocPoolMu.Unlock()
}
