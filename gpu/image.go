package gpu

import (
	"fmt"
	"runtime"

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

	img := newImage(newImageData(config), subImageConfigFromImageConfig(config))
	img.ownsData = true
	// TODO: runtime.AddCleanup
	return img
}

func NewImageFromVkImage(vkImage vk.Image, config ImageConfig) *Image {
	return newImage(newImageDataFromVkImage(vkImage, config), subImageConfigFromImageConfig(config))
}

func newImage(data *imageData, config subImageConfig) *Image {
	extent := minify3(data.extent, config.firstMip)
	formatClass := formatutil.Describe(config.format).Class
	baseFormatClass := formatutil.Describe(data.format).Class
	if formatClass != baseFormatClass {
		// Format classes differ, this can only be possible if we're
		// reinterpreting a compressed format as uncompressed, so block1 must be
		// 1, 1, 1.
		// TODO: check that formatClass is uncompressed, while baseFormatClass
		// is compressed instead of this hack
		if formatBlockExtent(config.format) != ([3]int{1, 1, 1}) {
			panic(fmt.Sprintf("cannot create a %v view of a %v class image", config.format, baseFormatClass))
		}
		extent = divByBlockExtentRoundUp(extent, data.format)
	}

	img := &Image{
		data:       data,
		descriptor: newImageDescriptor(data, config),
		dim:        config.dim,
		format:     config.format,
		bounds:     config.bounds(),
		extent:     vkExtent3DFromInt3(extent),
	}
	if img.descriptor != (ImageDescriptor{}) {
		img.cleanup = runtime.AddCleanup(img, destroyImageDescriptor, img.descriptor)
	}

	return img
}

// TODO: for multi-planar images we'd also want to specify aspect mask. Instead,
// we can pretend all images are multi-planar and specify *plane mask* (up to 3
// bits). Depth-stencil images always have depth be plane 0 and stencil plane 1.
// TODO: better parameter names
func (img *Image) SubImage(opts ...SubImageOption) *Image {
	config := subImageConfig{
		dim:        img.dim,
		format:     img.format,
		firstLayer: img.bounds.FirstLayer(),
		layers:     img.bounds.Layers(),
		firstMip:   img.bounds.FirstMip(),
		mips:       img.bounds.Mips(),
	}
	for _, opt := range opts {
		// TODO: switch over common impls so that we noescape things
		opt.apply(&config)
	}
	return newImage(img.data, config)
}

func (img *Image) Config() ImageConfig {
	return ImageConfig{
		dim:    img.dim,
		format: img.format,
		extent: int3FromVkExtent3D(img.extent),
		layers: img.bounds.Layers(),
		mips:   img.bounds.Mips(),
		usage:  img.data.usage, // TODO: we might need to coerce things here
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
