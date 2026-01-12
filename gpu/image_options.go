package gpu

import "worldspawn/gpu/vk"

// TODO: rename the methods in the option interfaces

type ImageOption interface{ apply2(config *imageConfig) }

// TODO: make this public too like ImageOption is
type SubImageOption interface{ apply(config *subImageConfig) }

// TODO: rename?
type ViewAs ImageDim

func (dim ViewAs) apply(config *subImageConfig) { config.Dim = ImageDim(dim) }

// TODO: rename back to Reinterpret?
type WithFormat Format

func (format WithFormat) apply(config *subImageConfig) { config.Format = Format(format) }

type WithLayers struct{ First, End int }

func (layers WithLayers) apply2(config *imageConfig) {
	config.Layers = layers.End - layers.First
}

func (layers WithLayers) apply(config *subImageConfig) {
	config.FirstLayer = config.FirstLayer + layers.First
	config.Layers = layers.End - layers.First
}

type WithMips struct{ First, End int }

func (mips WithMips) apply2(config *imageConfig) {
	config.Mips = mips.End - mips.First
}

func (mips WithMips) apply(config *subImageConfig) {
	config.FirstMip = config.FirstMip + mips.First
	config.Mips = mips.End - mips.First
}

type WithUsage vk.ImageUsageFlagBits

func (usage WithUsage) apply2(config *imageConfig) {
	config.Usages |= vk.ImageUsageFlags(usage)
}

type imageConfig struct {
	Dim    int
	Format Format
	Extent [3]int
	Layers int
	Mips   int
	Usages vk.ImageUsageFlags
}

func joinImageOptions(format vk.Format, extent []int, opts ...ImageOption) imageConfig {
	var conf imageConfig
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

type subImageConfig struct {
	Format     Format
	Dim        ImageDim
	FirstLayer int
	Layers     int
	FirstMip   int
	Mips       int
}

func joinSubImageOptions(conf *subImageConfig, opts ...SubImageOption) {
	// TODO: switch over common impls so that we noescape things

	for _, opt := range opts {
		// TODO: switch over common impls so that we noescape things
		opt.apply(conf)
	}
}
