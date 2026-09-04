package gpu

import "worldspawn/gpu/vk"

// TODO: make this more compact. i.e. use [3]uint32 for
// extent, uint8 for mips, etc.
type ImageConfig struct {
	dim    imageDim
	format vk.Format
	extent [maxDimensions]int
	mips   int
	layers int
}

func MakeImageConfig(format vk.Format, extent []int) ImageConfig {
	return ImageConfig{
		dim:    makeImageDim(len(extent)),
		format: format,
		extent: extent3(extent),
		mips:   1,
		layers: 1,
	}
}

// TODO: rename pls
// TODO: I wish we could just kill this
// TODO: replace this with an enum?
func (config ImageConfig) AsCube(cube bool) ImageConfig {
	// TODO: make this a method on ImageDim
	if cube {
		config.dim |= 0x80
	} else {
		config.dim &^= 0x80
	}
	return config
}

func (config ImageConfig) WithMips(mips int) ImageConfig {
	config.mips = mips
	return config
}

func (config ImageConfig) WithLayers(layers int) ImageConfig {
	config.layers = layers
	return config
}

func (config ImageConfig) IsCube() bool { return config.dim.isCube() }

func (config ImageConfig) Format() vk.Format { return config.format }

func (config ImageConfig) Extent() []int {
	d := config.dim.dimensions()
	return config.extent[:d]
}

func (config ImageConfig) Mips() int { return config.mips }

func (config ImageConfig) Layers() int { return config.layers }
