package gpu

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

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

	extent vk.Extent3D

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
	Extent    [3]int
	Layers    int
	MipLevels int
	Samples   uint32 // TODO: remove multisampled images altogether from our abstraction?
	Format    Format
	Usage     ImageUsage // TODO: a separate stencil usage? We could also pack usage for stencil into the high bits
}

func (config *ImageConfig) vkImageCreateInfo(queueFamilies []uint32, createInfo *vk.ImageCreateInfo) {
	flags := vk.ImageCreateFlags(0)

	flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_EXTENDED_USAGE_BIT)

	flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_MUTABLE_FORMAT_BIT)

	if config.Dim.vkImageType() == vk.IMAGE_TYPE_2D &&
		config.Extent[0] == config.Extent[1] &&
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
		Extent:                vkExtent3DFromInt3(config.Extent),
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

	must(vkFns.CreateImage(device, imageCreateInfo, nil, &base.vkImage))

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
		must(vkFns.AllocateMemory(device, &vk.MemoryAllocateInfo{
			SType:           vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
			AllocationSize:  vk.DeviceSize(size),
			MemoryTypeIndex: uint32(memoryTypeIndex),
		}, nil, &allocation.memory))
		base.memory = &allocation
	}

	must(vkFns.BindImageMemory2(device, 1, &vk.BindImageMemoryInfo{
		SType:        vk.STRUCTURE_TYPE_BIND_IMAGE_MEMORY_INFO,
		Image:        base.vkImage,
		Memory:       base.memory.memory,
		MemoryOffset: 0,
	}))

	base.dim = config.Dim
	base.extent = vkExtent3DFromInt3(config.Extent)
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

func importImage(config *ImageConfig, vkImage vk.Image) *Image {
	base := new(imageData)
	base.vkImage = vkImage
	base.dim = config.Dim
	base.extent = vkExtent3DFromInt3(config.Extent)
	base.layers = uint32(config.Layers)
	base.mipLevels = uint32(config.MipLevels)
	base.format = config.Format
	base.usage = config.Usage
	return newImage(
		base,
		config.Dim,
		config.Format,
		0, config.Layers,
		0, config.MipLevels)
}

func newImage(
	base *imageData,
	dim ImageDim,
	format Format,
	baseLayer, layers int,
	baseMipLevel, mipLevels int) *Image {
	extent := minify3(int3FromVkExtent3D(base.extent), baseMipLevel)
	formatClass := formatutil.Describe(format).Class
	baseFormatClass := formatutil.Describe(base.format).Class
	if formatClass != baseFormatClass {
		// Format classes differ, this can only be possible if we're
		// reinterpreting a compressed format as uncompressed, so block1 must be
		// 1, 1, 1.
		// TODO: check that formatClass is uncompressed, while baseFormatClass
		// is compressed instead of this hack
		if formatBlockExtent(format) != splat3(1) {
			panic(fmt.Sprintf("cannot create a %v view of a %v class image", format, baseFormatClass))
		}
		extent = int3DivRoundUp(extent, formatBlockExtent(base.format))
	}

	img := &Image{
		base:         base,
		descriptors:  newImageDescriptors(base, dim, format, baseLayer, layers, baseMipLevel, mipLevels),
		dim:          dim,
		format:       format,
		baseLayer:    uint32(baseLayer),
		layers:       uint32(layers),
		baseMipLevel: uint8(baseMipLevel),
		mipLevels:    uint8(mipLevels),
		extent:       vkExtent3DFromInt3(extent),
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

func (img *Image) Layers() int { return int(img.layers) }

func (img *Image) Extent() [3]int { return int3FromVkExtent3D(img.extent) }

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

func (img *Image) EnqueueInit(jq *JobQueue) {
	img.enqueueTransitionLayout(jq, vk.IMAGE_LAYOUT_UNDEFINED, vk.IMAGE_LAYOUT_GENERAL)
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
	// Stop the cleanup first.
	img.cleanup.Stop()

	img.descriptors.destroy()

	if img.ownsBase {
		img.base.destroy()
	}
}
