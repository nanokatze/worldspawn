package gpu

import (
	"fmt"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// Bits 0:1 specify number of dimensions, which is always at least 1.
// Bit 7 specifies cube flag. Only valid for 2D images.
//
// TODO: make this private
type ImageDim uint8

const (
	_ ImageDim = iota
	ImageDim1D
	ImageDim2D
	ImageDim3D
	ImageDimCube = ImageDim2D | 0x80
)

func (dim ImageDim) dimensions() int {
	return int(dim & 0b11)
}

func (dim ImageDim) vkImageType() vk.ImageType {
	switch dim.dimensions() {
	case 1:
		return vk.IMAGE_TYPE_1D
	case 2:
		return vk.IMAGE_TYPE_2D
	case 3:
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

type Image struct {
	data       *imageData
	descriptor ImageDescriptor

	dim    ImageDim
	format vk.Format
	bounds imageBounds

	// precomputed stuff

	extent vk.Extent3D

	// ownsData specifies whether this *Image owns the data and Destroy should
	// actually destroy it.
	//
	// TODO: remove once image memory imposes pressure on GC
	ownsData bool

	cleanup runtime.Cleanup
}

func NewImage(config ImageConfig) *Image {
	gpuInit()

	var pinner runtime.Pinner
	defer pinner.Unpin()

	imageCreateInfo := new(vk.ImageCreateInfo)
	imageCreateInfo.SType = vk.STRUCTURE_TYPE_IMAGE_CREATE_INFO
	// TODO: shove these into a function that operates on vk.ImageCreateInfo
	imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_MUTABLE_FORMAT_BIT)
	imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_EXTENDED_USAGE_BIT)
	// TODO: actually check if it's compressed instead of checking BlockExtent
	if formatutil.Describe(config.format).BlockExtent != (vk.Extent3D{Width: 1, Height: 1, Depth: 1}) {
		imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_BLOCK_TEXEL_VIEW_COMPATIBLE_BIT)
	}
	if config.dim == 2 && config.extent[0] == config.extent[1] && config.layers >= 6 {
		imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_CUBE_COMPATIBLE_BIT)
	}
	imageCreateInfo.ImageType = ImageDim(config.dim).vkImageType()
	imageCreateInfo.Format = config.format
	imageCreateInfo.Extent = vkExtent3DFromInt3(config.extent)
	imageCreateInfo.MipLevels = uint32(config.mips)
	imageCreateInfo.ArrayLayers = uint32(config.layers)
	imageCreateInfo.Samples = 1
	imageCreateInfo.Usage = config.usages
	imageCreateInfo.SharingMode = vk.SHARING_MODE_CONCURRENT
	imageCreateInfo.QueueFamilyIndexCount = uint32(len(topology.probe))
	imageCreateInfo.PQueueFamilyIndices = unsafe.SliceData(topology.probe)

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

	// println(fmt.Sprintf("image %v needs %v aligned to %v", config.Extent, requirements.Size, requirements.Alignment))

	size := roundUpDeviceAllocationSize(int(requirements.Size))

	var vkImage vk.Image
	must(vkFns.CreateImage(device, imageCreateInfo, nil, &vkImage))

	memoryTypeIndex := findMemoryTypeIndex(requirements.MemoryTypeBits, 0)

	var memory *deviceMemory
	{
		allocPoolMu.Lock()
		entries := allocPool[size]
		if len(entries) > 0 {
			memory = entries[len(entries)-1]
			allocPool[size] = entries[:len(entries)-1]
		}
		allocPoolMu.Unlock()
	}
	if memory == nil {
		var allocation deviceMemory
		allocation.size = size
		must(vkFns.AllocateMemory(device, &vk.MemoryAllocateInfo{
			SType:           vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
			AllocationSize:  vk.DeviceSize(size),
			MemoryTypeIndex: uint32(memoryTypeIndex),
		}, nil, &allocation.memory))
		memory = &allocation
	}

	must(vkFns.BindImageMemory2(device, 1, &vk.BindImageMemoryInfo{
		SType:        vk.STRUCTURE_TYPE_BIND_IMAGE_MEMORY_INFO,
		Image:        vkImage,
		Memory:       memory.memory,
		MemoryOffset: 0,
	}))

	img := newImageFromData(
		&imageData{
			vkImage: vkImage,

			format: config.format,
			extent: config.extent,
			layers: config.layers,
			mips:   config.mips,
			usages: config.usages,

			memory: memory,
		},
		config)
	img.ownsData = true
	// TODO: runtime.AddCleanup
	return img
}

func NewImageFromVkImage(vkImage vk.Image, config ImageConfig) *Image {
	return newImageFromData(
		&imageData{
			vkImage: vkImage,

			format: config.format,
			extent: config.extent,
			layers: config.layers,
			mips:   config.mips,
			usages: config.usages,
		},
		config)
}

func newImageFromData(data *imageData, config ImageConfig) *Image {
	return newImage(data,
		&subImageConfig{
			Dim:    config.dim,
			Format: config.format,
			Layers: config.layers,
			Mips:   config.mips,
		})
}

func newImage(data *imageData, config *subImageConfig) *Image {
	extent := minify3(data.extent, config.FirstMip)
	formatClass := formatutil.Describe(config.Format).Class
	baseFormatClass := formatutil.Describe(data.format).Class
	if formatClass != baseFormatClass {
		// Format classes differ, this can only be possible if we're
		// reinterpreting a compressed format as uncompressed, so block1 must be
		// 1, 1, 1.
		// TODO: check that formatClass is uncompressed, while baseFormatClass
		// is compressed instead of this hack
		if formatBlockExtent(config.Format) != ([3]int{1, 1, 1}) {
			panic(fmt.Sprintf("cannot create a %v view of a %v class image", config.Format, baseFormatClass))
		}
		extent = divByBlockExtentRoundUp(extent, data.format)
	}

	img := &Image{
		data:       data,
		descriptor: newImageDescriptor(data, config),
		dim:        config.Dim,
		format:     config.Format,
		bounds:     config.bounds(),
		extent:     vkExtent3DFromInt3(extent),
	}
	if img.descriptor != (ImageDescriptor{}) {
		img.cleanup = runtime.AddCleanup(img, destroyImageDescriptor, img.descriptor)
	}

	return img
}

type subImageConfig struct {
	Dim        ImageDim
	Format     vk.Format
	FirstLayer int
	Layers     int
	FirstMip   int
	Mips       int
}

type SubImageOption interface{ apply(config *subImageConfig) }

// TODO: rename?
// TODO: ViewAsCube{} variant
type ViewAs ImageDim

func (dim ViewAs) apply(config *subImageConfig) {
	// TODO: validation
	config.Dim = ImageDim(dim)
}

type Reinterpret vk.Format

func (format Reinterpret) apply(config *subImageConfig) {
	// TODO: validation
	config.Format = vk.Format(format)
}

// TODO: make a variant of this called WithSlices which would have to be used for 3D images?
type WithLayerRange [2]int

func (layers WithLayerRange) apply(config *subImageConfig) {
	// TODO: validation
	config.FirstLayer = config.FirstLayer + layers[0]
	config.Layers = layers[1] - layers[0]
}

type WithMipRange [2]int

func (mips WithMipRange) apply(config *subImageConfig) {
	// TODO: validation
	config.FirstMip = config.FirstMip + mips[0]
	config.Mips = mips[1] - mips[0]
}

func (conf *subImageConfig) join(opts ...SubImageOption) {
	// TODO: switch over common impls so that we noescape things

	for _, opt := range opts {
		// TODO: switch over common impls so that we noescape things
		opt.apply(conf)
	}
}

func (conf *subImageConfig) bounds() imageBounds {
	return makeImageBounds(formatutil.Aspects(conf.Format), conf.FirstLayer, conf.Layers, conf.FirstMip, conf.Mips)
}

// TODO: for multi-planar images we'd also want to specify aspect mask. Instead,
// we can pretend all images are multi-planar and specify *plane mask* (up to 3
// bits). Depth-stencil images always have depth be plane 0 and stencil plane 1.
// TODO: better parameter names
func (img *Image) SubImage(opts ...SubImageOption) *Image {
	conf := subImageConfig{
		Dim:        img.dim,
		Format:     img.format,
		FirstLayer: img.bounds.FirstLayer(),
		Layers:     img.bounds.Layers(),
		FirstMip:   img.bounds.FirstMip(),
		Mips:       img.bounds.Mips(),
	}
	conf.join(opts...)
	return newImage(img.data, &conf)
}

func (img *Image) Config() ImageConfig {
	return ImageConfig{
		dim:    img.dim,
		format: img.format,
		extent: int3FromVkExtent3D(img.extent),
		layers: img.bounds.Layers(),
		mips:   img.bounds.Mips(),
		usages: img.data.usages, // TODO: we might need to coerce things here
	}
}

// TODO: kill some of these in favor of (*Image).Config()

// TODO: kill this *definitely*.
func (img *Image) Dim() ImageDim { return img.dim }

func (img *Image) Format() vk.Format { return img.format }

func (img *Image) Extent() []int {
	tmp := int3FromVkExtent3D(img.extent)
	return tmp[:img.dim.dimensions()]
}

func (img *Image) Layers() int { return img.bounds.Layers() }

func (img *Image) Mips() int { return img.bounds.Mips() }

func (img *Image) EnqueueInit(jq *JobQueue) {
	img.EnqueueTransitionLayout(jq, vk.IMAGE_LAYOUT_UNDEFINED, vk.IMAGE_LAYOUT_GENERAL)
}

func (img *Image) Descriptor() ImageDescriptor {
	if img == nil {
		return ImageDescriptor{}
	}
	return img.descriptor
}

func (img *Image) VkImage() (vk.Image, vk.ImageSubresourceRange) {
	return img.data.vkImage, img.bounds.VkImageSubresourceRange(img.format)
}

// TODO: rename to "Free" or something like that and document that it's not
// necessary for the user to call it.
// TODO: make this robust to multiple destroy calls
func (img *Image) Destroy() {
	// Stop the cleanup first.
	img.cleanup.Stop()

	destroyImageDescriptor(img.descriptor)

	if img.ownsData {
		img.data.destroy()
	}
}

func formatBlockExtent(format vk.Format) [3]int {
	return int3FromVkExtent3D(formatutil.Describe(format).BlockExtent)
}
