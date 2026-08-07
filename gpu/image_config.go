package gpu

import (
	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

type ImageConfig struct {
	dim    imageDim
	format vk.Format
	extent [3]int
	layers int
	mips   int
}

func MakeImageConfig(format vk.Format, extent []int) ImageConfig {
	return ImageConfig{
		dim:    makeImageDim(len(extent)),
		format: format,
		extent: extent3(extent),
		layers: 1,
		mips:   1,
	}
}

// TODO: rename pls
// TODO: I wish we could just kill this
func (config ImageConfig) AsCube(cube bool) ImageConfig {
	// TODO: make this a method on ImageDim
	if cube {
		config.dim |= 0x80
	} else {
		config.dim &^= 0x80
	}
	return config
}

func (config ImageConfig) WithLayers(layers int) ImageConfig {
	config.layers = layers
	return config
}

func (config ImageConfig) WithMips(mips int) ImageConfig {
	config.mips = mips
	return config
}

func (config ImageConfig) Format() vk.Format { return config.format }

func (config ImageConfig) IsCube() bool { return config.dim.isCube() }

func (config ImageConfig) Extent() []int {
	d := config.dim.dimensions()
	return config.extent[:d]
}

func (config ImageConfig) Layers() int { return config.layers }

func (config ImageConfig) Mips() int { return config.mips }

type subImageConfig struct {
	dim        imageDim
	format     vk.Format
	firstLayer int
	layers     int
	firstMip   int
	mips       int
}

func subImageConfigFromImageConfig(config ImageConfig) subImageConfig {
	return subImageConfig{
		dim:    config.dim,
		format: config.format,
		layers: config.layers,
		mips:   config.mips,
	}
}

func (config subImageConfig) bounds() imageBounds {
	return makeImageBounds(formatutil.Aspects(config.format), config.firstLayer, config.layers, config.firstMip, config.mips)
}

type SubImageOption interface {
	// TODO: this should also take the *Image so that we can perform validation
	// during construction. We could also supply that info through
	// subImageConfig
	apply(config *subImageConfig)
}

// TODO: validation

// TODO: make all of these be SubImageOption constructors instead of plain types

// TODO: rename?
// TODO: ViewAsCube{} variant
type ViewAs int

func (dim ViewAs) apply(config *subImageConfig) {
	config.dim = makeImageDim(int(dim))
}

type Reinterpret vk.Format

func (format Reinterpret) apply(config *subImageConfig) {
	// TODO: validation
	config.format = vk.Format(format)
}

// TODO: make a variant of this called SliceSlices to be used for 3D images
type SliceLayers [2]int

func (layers SliceLayers) apply(config *subImageConfig) {
	// TODO: validation
	config.firstLayer = config.firstLayer + layers[0]
	config.layers = layers[1] - layers[0]
}

type SliceMips [2]int

func (mips SliceMips) apply(config *subImageConfig) {
	// TODO: validation
	config.firstMip = config.firstMip + mips[0]
	config.mips = mips[1] - mips[0]
}
