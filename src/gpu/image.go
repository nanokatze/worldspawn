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

var imageDescAllocHint int64
var imageDescAlloc = newSlotAlloc(1e6) // allocate either at NewImage() or gpuInit()
var imageViews = make([]vk.ImageView, 1e6)

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

func (base *imageData) destroy() {
	vkFns.DestroyImage(device, base.vkImage, nil)

	allocPoolMu.Lock()
	allocPool[base.memory.size] = append(allocPool[base.memory.size], base.memory)
	allocPoolMu.Unlock()
}

type imageDescriptors struct {
	sampling, loadStore uint32
}

func (descriptors imageDescriptors) destroy() {
	vkFns.DestroyImageView(device, imageViews[descriptors.sampling], nil)
	imageDescAlloc.Free(int(descriptors.sampling))
	vkFns.DestroyImageView(device, imageViews[descriptors.loadStore], nil)
	imageDescAlloc.Free(int(descriptors.loadStore))
}

type Image struct {
	base        *imageData // TODO: rename to data?
	descriptors imageDescriptors

	dim          ImageDim
	format       Format // see SubImage
	baseLayer    uint32
	layers       uint32
	baseMipLevel uint8 // TODO: compress these into a single uint64
	mipLevels    uint8

	// precomputed stuff

	extent Int3 // TODO: change to Extent3D for a more compact representation?

	// ownsBase specifies whether this *Image owns the data and Destroy should
	// actually destroy it.
	//
	// TODO: remove once image memory imposes pressure on GC
	ownsBase bool

	cleanup runtime.Cleanup
}

// TODO: do we need any other fields here?
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

func (config *ImageConfig) vkImageCreateInfo(queueFamilies []uint32, createInfo *vk.ImageCreateInfo) {
	flags := vk.ImageCreateFlags(0)

	flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_EXTENDED_USAGE_BIT)

	flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_MUTABLE_FORMAT_BIT)

	if config.Dim == ImageDim2D &&
		config.Extent.X == config.Extent.Y &&
		config.Layers >= 6 &&
		config.Samples == 1 {
		flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_CUBE_COMPATIBLE_BIT)
	}

	// TODO: actually check if it's compressed instead of checking BlockExtent
	if formatutil.Describe(config.Format).BlockExtent != (vk.Extent3D{Width: 1, Height: 1, Depth: 1}) {
		flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_BLOCK_TEXEL_VIEW_COMPATIBLE_BIT)
	}

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

func NewImage(config *ImageConfig) *Image {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	base := new(imageData)

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

	if err := vkFns.CreateImage(device, imageCreateInfo, nil, &base.vkImage); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateImage: %v", err))
	}

	memoryTypeIndex := findMemoryTypeIndex(requirements.MemoryTypeBits, 0)

	{
		allocPoolMu.Lock()
		entries := allocPool[size]
		if len(entries) > 0 {
			base.memory = entries[len(entries)-1]
			allocPool[size] = entries[:len(entries)-1]
		}
		allocPoolMu.Unlock()
	}
	if base.memory == nil {
		var allocation deviceMemory
		allocation.size = size
		if err := vkFns.AllocateMemory(device, &vk.MemoryAllocateInfo{
			SType:           vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
			AllocationSize:  vk.DeviceSize(size),
			MemoryTypeIndex: uint32(memoryTypeIndex),
		}, nil, &allocation.memory); err != nil {
			panic(fmt.Sprintf("gpu: vkAllocateMemory: %v", err))
		}
		base.memory = &allocation
	}

	if err := vkFns.BindImageMemory2(device, 1, &vk.BindImageMemoryInfo{
		SType:        vk.STRUCTURE_TYPE_BIND_IMAGE_MEMORY_INFO,
		Image:        base.vkImage,
		Memory:       base.memory.memory,
		MemoryOffset: 0,
	}); err != nil {
		panic(fmt.Sprintf("gpu: vkBindImageMemory2: %v", err))
	}

	base.dim = config.Dim
	base.extent = config.Extent
	base.layers = uint32(config.Layers)
	base.mipLevels = uint32(config.MipLevels)
	base.format = config.Format
	base.usage = config.Usage

	/*
		runtime.AddCleanup(imageData,
			func(vkImage vk.Image) {

			},
			struct{}{})
	*/

	tmp := newImage(
		base,
		base.dim,
		base.format,
		0, int(base.layers),
		0, int(base.mipLevels))
	tmp.ownsBase = true
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

	img := &Image{
		base:         base,
		dim:          dim,
		format:       format,
		baseLayer:    uint32(baseLayer),
		layers:       uint32(layers),
		baseMipLevel: uint8(baseMipLevel),
		mipLevels:    uint8(mipLevels),
		extent:       extent,
	}

	formatFeatures := getFormatProps(format).OptimalTilingFeatures

	const formatFeatureMaskImageSampling = vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_SAMPLED_IMAGE_BIT)
	sampling := base.usage&ImageUsageSampling != 0 &&
		formatFeatures&formatFeatureMaskImageSampling == formatFeatureMaskImageSampling

	const formatFeatureMaskImageLoadStore = vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_IMAGE_BIT) |
		vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_READ_WITHOUT_FORMAT_BIT) |
		vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_WRITE_WITHOUT_FORMAT_BIT)
	loadStore := base.usage&ImageUsageLoadStore != 0 &&
		formatFeatures&formatFeatureMaskImageLoadStore == formatFeatureMaskImageLoadStore

	var imageViewCreateInfo *vk.ImageViewCreateInfo
	if sampling || loadStore {
		usage := vk.ImageUsageFlags(0)
		if sampling {
			usage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT)
		}
		if loadStore {
			usage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT)
		}

		imageViewCreateInfo = &vk.ImageViewCreateInfo{
			SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
			PNext: unsafe.Pointer(&vk.ImageViewUsageCreateInfo{
				SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO,
				Usage: usage,
			}),
			Image:            img.base.vkImage,
			ViewType:         img.dim.vkImageViewType(),
			Format:           img.format,
			SubresourceRange: img.vkImageSubresourceRange(),
		}
	}

	if sampling {
		img.descriptors.sampling = uint32(imageDescAlloc.Alloc(&imageDescAllocHint))

		var vkView vk.ImageView
		if err := vkFns.CreateImageView(device, imageViewCreateInfo, nil, &vkView); err != nil {
			panic(fmt.Sprintf("gpu: vkCreateImageView: %v", err))
		}
		imageViews[img.descriptors.sampling] = vkView

		vkFns.UpdateDescriptorSets(device,
			1, &vk.WriteDescriptorSet{
				SType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
				DstSet:          descriptorSet,
				DstBinding:      1,
				DstArrayElement: uint32(img.descriptors.sampling),
				DescriptorCount: 1,
				DescriptorType:  vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE,
				PImageInfo: &vk.DescriptorImageInfo{
					ImageView:   vkView,
					ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
				},
			},
			0, nil)
	}

	if loadStore {
		img.descriptors.loadStore = uint32(imageDescAlloc.Alloc(&imageDescAllocHint))

		var vkView vk.ImageView
		if err := vkFns.CreateImageView(device, imageViewCreateInfo, nil, &vkView); err != nil {
			panic(fmt.Sprintf("gpu: vkCreateImageView: %v", err))
		}
		imageViews[img.descriptors.loadStore] = vkView

		vkFns.UpdateDescriptorSets(device,
			1, &vk.WriteDescriptorSet{
				SType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
				DstSet:          descriptorSet,
				DstBinding:      2,
				DstArrayElement: uint32(img.descriptors.loadStore),
				DescriptorCount: 1,
				DescriptorType:  vk.DESCRIPTOR_TYPE_STORAGE_IMAGE,
				PImageInfo: &vk.DescriptorImageInfo{
					ImageView:   vkView,
					ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
				},
			},
			0, nil)
	}

	if img.descriptors != (imageDescriptors{}) {
		img.cleanup = runtime.AddCleanup(img, imageDescriptors.destroy, img.descriptors)
	}

	return img
}

// Format specifies what format to reinterpret this image
//
// TODO: for multi-planar images we'd also want to specify aspect mask. Instead,
// we can pretend all images are multi-planar and specify *plane mask* (up to 3
// bits). Depth-stencil images always have depth be plane 0 and stencil plane 1.
// TODO: better parameter names
func (img *Image) SubImage(
	dim ImageDim,
	format Format,
	baseLayer, endLayer int,
	baseMipLevel, endMipLevel int) *Image {
	// TODO: SubImage-specific validation

	return newImage(
		img.base,
		dim,
		format,
		int(img.baseLayer)+baseLayer, endLayer-baseLayer,
		int(img.baseMipLevel)+baseMipLevel, endMipLevel-baseMipLevel)
}

func (img *Image) Dim() ImageDim { return img.dim }

func (img *Image) Format() Format { return img.format }

func (img *Image) Extent() Int3 { return img.extent }

func (img *Image) EnqueueInit(jq *JobQueue) {
	img.enqueueTransitionLayout(jq, vk.IMAGE_LAYOUT_UNDEFINED, vk.IMAGE_LAYOUT_GENERAL)
}

// TODO: move to its own file I guess. Or idk.
type transitionImageLayoutJob struct {
	imageData        *imageData
	subresourceRange vk.ImageSubresourceRange
	oldLayout        vk.ImageLayout
	newLayout        vk.ImageLayout
}

func (img *Image) enqueueTransitionLayout(jq *JobQueue, oldLayout, newLayout vk.ImageLayout) {
	jq.Enqueue(&transitionImageLayoutJob{
		imageData:        img.base,
		subresourceRange: img.vkImageSubresourceRange(),
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

func (img *Image) SamplingDescriptor() SamplingView {
	if img.descriptors.sampling == 0 {
		panic("no descriptor")
	}
	return SamplingView{img.descriptors.sampling}
}

func (img *Image) LoadStoreDescriptor() StorageView {
	if img.descriptors.loadStore == 0 {
		panic("no descriptor")
	}
	return StorageView{img.descriptors.loadStore}
}

func (img *Image) vkImageSubresourceRange() vk.ImageSubresourceRange {
	return vk.ImageSubresourceRange{
		AspectMask:     formatutil.Aspects(img.format),
		BaseMipLevel:   uint32(img.baseMipLevel),
		LevelCount:     uint32(img.mipLevels),
		BaseArrayLayer: img.baseLayer,
		LayerCount:     img.layers,
	}
}

// TODO: rename to "Free" or something like that and document that it's not
// necessary for the user to call it.
func (img *Image) Destroy() {
	img.descriptors.destroy()

	if img.ownsBase {
		img.base.destroy()
	}

	img.cleanup.Stop()
}
