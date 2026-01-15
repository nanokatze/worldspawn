package gpu

import "worldspawn/gpu/vk"

// TODO: use an even more compact representation
// TODO: this also needs an aspect/plane mask eventually
type subresourceRange struct {
	aspects    vk.ImageAspectFlags // TODO: replace with our own mask
	firstLayer uint32
	layers     uint32
	firstMip   uint8
	mips       uint8
}

func makeSubresourceRange(aspects vk.ImageAspectFlags, firstLayer, layers, firstMip, mips int) subresourceRange {
	return subresourceRange{
		aspects:    aspects,
		firstLayer: uint32(firstLayer),
		layers:     uint32(layers),
		firstMip:   uint8(firstMip),
		mips:       uint8(mips),
	}
}

func (sr subresourceRange) FirstLayer() int { return int(sr.firstLayer) }

func (sr subresourceRange) Layers() int { return int(sr.layers) }

func (sr subresourceRange) FirstMip() int { return int(sr.firstMip) }

func (sr subresourceRange) Mips() int { return int(sr.mips) }

func (sr subresourceRange) VkImageSubresourceLayers(format vk.Format) vk.ImageSubresourceLayers {
	return vk.ImageSubresourceLayers{
		AspectMask:     sr.aspects,
		MipLevel:       uint32(sr.FirstMip()),
		BaseArrayLayer: uint32(sr.FirstLayer()),
		LayerCount:     uint32(sr.Layers()),
	}
}

func (sr subresourceRange) VkImageSubresourceRange(format vk.Format) vk.ImageSubresourceRange {
	return vk.ImageSubresourceRange{
		AspectMask:     sr.aspects,
		BaseMipLevel:   uint32(sr.FirstMip()),
		LevelCount:     uint32(sr.Mips()),
		BaseArrayLayer: uint32(sr.FirstLayer()),
		LayerCount:     uint32(sr.Layers()),
	}
}
