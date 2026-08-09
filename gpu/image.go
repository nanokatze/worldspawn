package gpu

import (
	"fmt"
	"runtime"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

const maxDimensions = 3

// Bits 0:6 indicate the number of dimensions, which is always at least 1.
// Bit 7 indicates the cube flag. Only valid for 2D images.
type imageDim uint8

func makeImageDim(dimensions int) imageDim {
	if !(1 <= dimensions && dimensions <= maxDimensions) {
		panic("bad number of dimensions")
	}
	return imageDim(uint8(dimensions))
}

func (dim imageDim) dimensions() int {
	return int(dim &^ 0x80)
}

func (dim imageDim) isCube() bool { return dim&0x80 != 0 }

func (dim imageDim) vkImageType() vk.ImageType {
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

func (dim imageDim) vkImageViewType() vk.ImageViewType {
	switch dim {
	case 1:
		return vk.IMAGE_VIEW_TYPE_1D_ARRAY
	case 2:
		return vk.IMAGE_VIEW_TYPE_2D_ARRAY
	case 2 | 0x80:
		return vk.IMAGE_VIEW_TYPE_CUBE_ARRAY
	case 3:
		return vk.IMAGE_VIEW_TYPE_3D
	default:
		panic("unreachable")
	}
}

type Image struct {
	data       *imageData
	descriptor ImageDescriptor

	dim    imageDim
	format vk.Format
	bounds imageBounds

	// precomputed stuff

	extent vk.Extent3D // TODO: replace with [3]uint32

	// ownsData specifies whether this *Image owns the data and Destroy should
	// actually destroy it.
	//
	// TODO: remove once image memory imposes pressure on GC
	ownsData bool

	cleanup runtime.Cleanup
}

func NewImage(config ImageConfig, usage vk.ImageUsageFlags) *Image {
	gpuInit()

	img := newImage(newImageData(config, usage), subImageConfigFromImageConfig(config))
	img.ownsData = true
	// TODO: runtime.AddCleanup
	return img
}

func NewImageFromVkImage(config ImageConfig, usage vk.ImageUsageFlags, vkImage vk.Image) *Image {
	return newImage(newImageDataFromVkImage(config, usage, vkImage), subImageConfigFromImageConfig(config))
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
		if formatutil.Describe(config.format).BlockExtent != ([3]int{1, 1, 1}) {
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
	}
}

// TODO: kill some of these in favor of (*Image).Config()

func (img *Image) Format() vk.Format { return img.format }

func (img *Image) Extent() []int {
	d := img.dim.dimensions()
	tmp := int3FromVkExtent3D(img.extent)
	return tmp[:d]
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

func (img *Image) VkImage() (vk.Image, vk.ImageViewType, vk.ImageSubresourceRange) {
	return img.data.vkImage, img.dim.vkImageViewType(), img.bounds.VkImageSubresourceRange(img.format)
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
