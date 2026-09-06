package gpu

import (
	"fmt"
	"runtime"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

type Image struct {
	data       *imageData
	descriptor ImageDescriptor

	dim    imageDim
	format vk.Format
	bounds imageBounds

	// precomputed stuff

	extent [3]uint32

	// ownsData specifies whether this *Image owns the data and Destroy should
	// actually destroy it.
	//
	// TODO: remove once image memory imposes pressure on GC
	ownsData bool

	cleanup runtime.Cleanup
}

type newImageOptions struct {
	usage vk.ImageUsageFlags

	vkImage vk.Image
}

type NewImageOption func(*newImageOptions)

func ImageWithUsage(usage vk.ImageUsageFlagBits) NewImageOption {
	return func(opts *newImageOptions) {
		opts.usage = vk.ImageUsageFlags(usage)
	}
}

func ImportVkImage(vkImage vk.Image) NewImageOption {
	return func(opts *newImageOptions) {
		opts.vkImage = vkImage
	}
}

func NewImage(config ImageConfig, opts ...NewImageOption) *Image {
	GPUInit()

	var optStruct newImageOptions
	for _, opt := range opts {
		opt(&optStruct)
	}

	data := makeImageData(config, &optStruct)

	img := newImage(data,
		&subImageOptions{
			dim:    config.dim,
			format: config.format,
			mips:   config.mips,
			layers: config.layers,
		})
	img.ownsData = true
	// TODO: runtime.AddCleanup
	return img
}

type SubImageOption interface {
	// TODO: this should also take the *Image so that we can perform validation
	// during construction.
	apply(opts *subImageOptions)
}

// TODO: validation

// TODO: make all of these be SubImageOption constructors instead of plain types

// TODO: rename?
// TODO: ViewAsCube{} variant
type ViewAs int

func (dim ViewAs) apply(opts *subImageOptions) {
	opts.dim = makeImageDim(int(dim))
}

type Reinterpret vk.Format

func (format Reinterpret) apply(opts *subImageOptions) {
	// TODO: validation
	opts.format = vk.Format(format)
}

type SliceMips [2]int

func (mips SliceMips) apply(opts *subImageOptions) {
	// TODO: validation
	opts.firstMip = opts.firstMip + mips[0]
	opts.mips = mips[1] - mips[0]
}

// TODO: make a variant of this called SliceSlices to be used for 3D images
type SliceLayers [2]int

func (layers SliceLayers) apply(opts *subImageOptions) {
	// TODO: validation
	opts.firstLayer = opts.firstLayer + layers[0]
	opts.layers = layers[1] - layers[0]
}

// TODO: for multi-planar images we'd also want to specify aspect mask. Instead,
// we can pretend all images are multi-planar and specify *plane mask* (up to 3
// bits). Depth-stencil images always have depth be plane 0 and stencil plane 1.
// TODO: better parameter names
func (img *Image) SubImage(opts ...SubImageOption) *Image {
	optStruct := subImageOptions{
		dim:        img.dim,
		format:     img.format,
		firstMip:   img.bounds.FirstMip(),
		mips:       img.bounds.Mips(),
		firstLayer: img.bounds.FirstLayer(),
		layers:     img.bounds.Layers(),
	}
	for _, opt := range opts {
		// TODO: switch over common impls so that we can noescape things
		opt.apply(&optStruct)
	}
	return newImage(img.data, &optStruct)
}

type subImageOptions struct {
	dim        imageDim
	format     vk.Format
	firstMip   int
	mips       int
	firstLayer int
	layers     int
}

func (opts subImageOptions) bounds() imageBounds {
	return makeImageBounds(formatutil.Aspects(opts.format), opts.firstMip, opts.mips, opts.firstLayer, opts.layers)
}

func newImage(data *imageData, opts *subImageOptions) *Image {
	extent := minify3(data.config.extent, opts.firstMip)

	baseFormatClass := formatutil.Describe(data.config.format).Class
	viewFormatClass := formatutil.Describe(opts.format).Class
	if viewFormatClass != baseFormatClass {
		// Format classes differ, this can only be possible if we're
		// reinterpreting a compressed format as uncompressed, so block1 must be
		// 1, 1, 1.
		// TODO: check that viewFormatClass is uncompressed, while
		// baseFormatClass is compressed instead of this hack
		if formatutil.Describe(opts.format).BlockExtent != pointOne() {
			panic(fmt.Sprintf("cannot create a %v view of a %v class image", opts.format, baseFormatClass))
		}
		extent = divByBlockExtentRoundUp(extent, data.config.format)
	}

	img := &Image{
		data:       data,
		descriptor: newImageDescriptor(data, opts),
		dim:        opts.dim,
		format:     opts.format,
		bounds:     opts.bounds(),
		extent: [3]uint32{ // TODO: outline into function
			uint32(extent[0]),
			uint32(extent[1]),
			uint32(extent[2]),
		},
	}
	if img.descriptor != (ImageDescriptor{}) {
		img.cleanup = runtime.AddCleanup(img, cleanupImageDescriptor, img.descriptor)
	}
	return img
}

// TODO: add (*Image).Supports(usage)

func (img *Image) Config() ImageConfig {
	return ImageConfig{
		dim:    img.dim,
		format: img.format,
		extent: point{ // TODO: outline into function
			int(img.extent[0]),
			int(img.extent[1]),
			int(img.extent[2]),
		},
		mips:   img.bounds.Mips(),
		layers: img.bounds.Layers(),
	}
}

// Equivalent to img.Config().Format()
func (img *Image) Format() vk.Format { return img.Config().Format() }

// Equivalent to img.Config().Extent()
func (img *Image) Extent() []int { return img.Config().Extent() }

// Equivalent to img.Config().Mips()
func (img *Image) Mips() int { return img.Config().Mips() }

// Equivalent to img.Config().Layers()
func (img *Image) Layers() int { return img.Config().Layers() }

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

	cleanupImageDescriptor(img.descriptor)

	if img.ownsData {
		img.data.destroy()
	}
}
