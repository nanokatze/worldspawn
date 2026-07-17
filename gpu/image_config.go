package gpu

import "worldspawn/gpu/vk"

// TODO: use a more compact representation
// TODO: allow cubes
type ImageConfig struct {
	dim    ImageDim
	format vk.Format
	extent [3]int
	layers int
	mips   int
	usages vk.ImageUsageFlags
}

func MakeImageConfig(format vk.Format, extent []int) ImageConfig {
	return ImageConfig{
		dim:    ImageDim(len(extent)),
		format: format,
		extent: extent3(extent),
		layers: 1,
		mips:   1,
		usages: vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_DST_BIT) | vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_SRC_BIT),
	}
}

func (config ImageConfig) WithLayers(layers int) ImageConfig {
	config.layers = layers
	return config
}

func (config ImageConfig) WithMips(mips int) ImageConfig {
	config.mips = mips
	return config
}

func (config ImageConfig) WithUsage(usage vk.ImageUsageFlagBits) ImageConfig {
	config.usages |= vk.ImageUsageFlags(usage)
	return config
}

func (config ImageConfig) Format() vk.Format { return config.format }

func (config ImageConfig) Extent() []int {
	return config.extent[:config.dim.dimensions()]
}

func (config ImageConfig) Layers() int { return config.layers }

func (config ImageConfig) Mips() int { return config.mips }

func (config ImageConfig) Usages() vk.ImageUsageFlags { return config.usages }
