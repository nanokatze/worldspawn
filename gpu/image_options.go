package gpu

import (
	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: rename the methods in the option interfaces

type ImageOption interface{ apply2(config *ImageConfig) }

// TODO: make this public too like ImageOption is
type SubImageOption interface{ apply(config *subImageConfig) }

// TODO: rename?
type ViewAs ImageDim

func (dim ViewAs) apply(config *subImageConfig) { config.Dim = ImageDim(dim) }

// TODO: rename back to Reinterpret?
type WithFormat vk.Format

func (format WithFormat) apply(config *subImageConfig) { config.Format = vk.Format(format) }

type WithLayers struct{ First, End int }

func (layers WithLayers) apply2(config *ImageConfig) {
	config.Layers = layers.End - layers.First
}

func (layers WithLayers) apply(config *subImageConfig) {
	config.FirstLayer = config.FirstLayer + layers.First
	config.Layers = layers.End - layers.First
}

type WithMips struct{ First, End int }

func (mips WithMips) apply2(config *ImageConfig) {
	config.Mips = mips.End - mips.First
}

func (mips WithMips) apply(config *subImageConfig) {
	config.FirstMip = config.FirstMip + mips.First
	config.Mips = mips.End - mips.First
}

type WithUsage vk.ImageUsageFlagBits

func (usage WithUsage) apply2(config *ImageConfig) {
	config.Usages |= vk.ImageUsageFlags(usage)
}

type ImageConfig struct {
	Dim    int
	Format vk.Format
	Extent [3]int
	Layers int
	Mips   int
	Usages vk.ImageUsageFlags
}

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

type subImageConfig struct {
	Dim        ImageDim
	Format     vk.Format
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

func (conf *subImageConfig) bounds() imageBounds {
	return makeImageBounds(formatutil.Aspects(conf.Format), conf.FirstLayer, conf.Layers, conf.FirstMip, conf.Mips)
}
