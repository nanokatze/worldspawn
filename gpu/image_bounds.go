package gpu

import "worldspawn/gpu/vk"

// TODO: use an even more compact representation
// TODO: this also needs an aspect/plane mask eventually
type imageBounds struct {
	aspects    vk.ImageAspectFlags // TODO: replace with our own mask
	firstLayer uint32
	layers     uint32
	firstMip   uint8
	mips       uint8
}

func makeImageBounds(aspects vk.ImageAspectFlags, firstLayer, layers, firstMip, mips int) imageBounds {
	return imageBounds{
		aspects:    aspects,
		firstLayer: uint32(firstLayer),
		layers:     uint32(layers),
		firstMip:   uint8(firstMip),
		mips:       uint8(mips),
	}
}

func (bounds imageBounds) FirstLayer() int { return int(bounds.firstLayer) }

func (bounds imageBounds) Layers() int { return int(bounds.layers) }

func (bounds imageBounds) FirstMip() int { return int(bounds.firstMip) }

func (bounds imageBounds) Mips() int { return int(bounds.mips) }

// TODO: introduce a function just for the bounds also

func (bounds imageBounds) VkImageSubresourceLayers(format vk.Format) vk.ImageSubresourceLayers {
	return vk.ImageSubresourceLayers{
		AspectMask:     bounds.aspects,
		MipLevel:       uint32(bounds.FirstMip()),
		BaseArrayLayer: uint32(bounds.FirstLayer()),
		LayerCount:     uint32(bounds.Layers()),
	}
}

func (bounds imageBounds) VkImageSubresourceRange(format vk.Format) vk.ImageSubresourceRange {
	return vk.ImageSubresourceRange{
		AspectMask:     bounds.aspects,
		BaseMipLevel:   uint32(bounds.FirstMip()),
		LevelCount:     uint32(bounds.Mips()),
		BaseArrayLayer: uint32(bounds.FirstLayer()),
		LayerCount:     uint32(bounds.Layers()),
	}
}
