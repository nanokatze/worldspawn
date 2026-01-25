package gpu

import (
	"fmt"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: get rid of this in favor of plain int + a flag on SubImage.
type ImageDim int8

const (
	_ ImageDim = iota
	ImageDim1D
	ImageDim2D
	ImageDim3D
	ImageDimCube = -ImageDim2D // TODO: try and kill this
)

func (dim ImageDim) dimensions() int {
	if dim < 0 {
		return -int(dim)
	}
	return int(dim)
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
	data        *imageData
	descriptors ImageDescriptors

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

type ImageConfig struct {
	Dim    int
	Format vk.Format
	Extent [3]int
	Layers int
	Mips   int
	Usages vk.ImageUsageFlags
}

type ImageOption interface{ apply2(config *ImageConfig) }

type WithLayers2 int

func (layers WithLayers2) apply2(config *ImageConfig) {
	config.Layers = int(layers)
}

type WithMips2 int

func (mips WithMips2) apply2(config *ImageConfig) {
	config.Mips = int(mips)
}

// TODO: rename?
func JoinImageOptions(format vk.Format, extent []int, opts ...ImageOption) ImageConfig {
	var conf ImageConfig
	conf.Dim = len(extent)
	conf.Format = format
	conf.Extent = extent3(extent)
	conf.Layers = 1
	conf.Mips = 1
	conf.Usages = vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_DST_BIT) | vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_SRC_BIT)
	for _, opt := range opts {
		// TODO: switch over types so that we don't leak the config
		opt.apply2(&conf)
	}
	return conf
}

func NewImage(format vk.Format, extent []int, opts ...ImageOption) *Image {
	gpuInit()

	conf := JoinImageOptions(format, extent, opts...)

	var pinner runtime.Pinner
	defer pinner.Unpin()

	imageCreateInfo := new(vk.ImageCreateInfo)
	imageCreateInfo.SType = vk.STRUCTURE_TYPE_IMAGE_CREATE_INFO
	// TODO: shove these into a function that operates on vk.ImageCreateInfo
	imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_MUTABLE_FORMAT_BIT)
	imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_EXTENDED_USAGE_BIT)
	// TODO: actually check if it's compressed instead of checking BlockExtent
	if formatutil.Describe(conf.Format).BlockExtent != (vk.Extent3D{Width: 1, Height: 1, Depth: 1}) {
		imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_BLOCK_TEXEL_VIEW_COMPATIBLE_BIT)
	}
	if conf.Dim == 2 && conf.Extent[0] == conf.Extent[1] && conf.Layers >= 6 {
		imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_CUBE_COMPATIBLE_BIT)
	}
	imageCreateInfo.ImageType = ImageDim(conf.Dim).vkImageType()
	imageCreateInfo.Format = conf.Format
	imageCreateInfo.Extent = vkExtent3DFromInt3(conf.Extent)
	imageCreateInfo.MipLevels = uint32(conf.Mips)
	imageCreateInfo.ArrayLayers = uint32(conf.Layers)
	imageCreateInfo.Samples = 1
	imageCreateInfo.Usage = conf.Usages
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

	img := newImageFromData(&imageData{
		vkImage: vkImage,

		dim:    conf.Dim,
		format: conf.Format,
		extent: conf.Extent,
		layers: conf.Layers,
		mips:   conf.Mips,
		usages: conf.Usages,

		memory: memory,
	})
	img.ownsData = true
	// TODO: runtime.AddCleanup
	return img
}

func NewImageFromVkImage(vkImage vk.Image, format vk.Format, extent []int, opts ...ImageOption) *Image {
	conf := JoinImageOptions(format, extent, opts...)

	return newImageFromData(&imageData{
		vkImage: vkImage,

		dim:    conf.Dim,
		format: conf.Format,
		extent: conf.Extent,
		layers: conf.Layers,
		mips:   conf.Mips,
		usages: conf.Usages,
	})
}

func newImageFromData(data *imageData) *Image {
	return newImage(data,
		&subImageConfig{
			Dim:    ImageDim(data.dim),
			Format: data.format,
			Layers: data.layers,
			Mips:   data.mips,
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
		data:        data,
		descriptors: newImageDescriptors(data, config),
		dim:         config.Dim,
		format:      config.Format,
		bounds:      config.bounds(),
		extent:      vkExtent3DFromInt3(extent),
	}
	if img.descriptors != (ImageDescriptors{}) {
		img.cleanup = runtime.AddCleanup(img, destroyImageDescriptors, img.descriptors)
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

// TODO: make this public too like ImageOption is
type SubImageOption interface{ apply(config *subImageConfig) }

// TODO: rename?
type ViewAs ImageDim

func (dim ViewAs) apply(config *subImageConfig) { config.Dim = ImageDim(dim) }

// TODO: separate WithLayers and WithMips ImageOptions pls? So that we don't
// need to specify First/End but just no. of mips.

// TODO: rename back to Reinterpret?
type WithFormat vk.Format

func (format WithFormat) apply(config *subImageConfig) { config.Format = vk.Format(format) }

// TODO: make a variant of this called WithSlices which would have to be used for 3D images?
type WithLayers struct{ First, End int }

func (layers WithLayers) apply(config *subImageConfig) {
	config.FirstLayer = config.FirstLayer + layers.First
	config.Layers = layers.End - layers.First
}

type WithMips struct{ First, End int }

func (mips WithMips) apply(config *subImageConfig) {
	config.FirstMip = config.FirstMip + mips.First
	config.Mips = mips.End - mips.First
}

type WithUsage vk.ImageUsageFlagBits

func (usage WithUsage) apply2(config *ImageConfig) {
	config.Usages |= vk.ImageUsageFlags(usage)
}

// TODO: make it a method on the *subImageConfig?
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

// Format specifies what format to reinterpret this image
//
// TODO: make this use vararg options (implemented in terms of interface so that
// we can handle things in a switch and avoid escapes)
//
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

func (img *Image) Descriptors() ImageDescriptors {
	if img == nil {
		return ImageDescriptors{}
	}
	return img.descriptors
}

// deprecated
func (img *Image) SamplingDescriptor() SamplingView {
	if img.descriptors.bits&(1<<20) == 0 {
		panic("no descriptor")
	}
	return SamplingView{uint32(img.descriptors.bits&(1<<20-1)) + 0}
}

// deprecated
func (img *Image) LoadStoreDescriptor() uint32 {
	if img.descriptors.bits&(1<<21) == 0 {
		panic("no descriptor")
	}
	return uint32(img.descriptors.bits&(1<<20-1)) + 1
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

	destroyImageDescriptors(img.descriptors)

	if img.ownsData {
		img.data.destroy()
	}
}

func formatBlockExtent(format vk.Format) [3]int {
	return int3FromVkExtent3D(formatutil.Describe(format).BlockExtent)
}
