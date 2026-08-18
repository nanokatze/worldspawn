package gpu

import (
	"runtime"

	"worldspawn/gpu/vk"
)

type ImageSampler struct{ bits uint32 }

var samplerHint int64

// TODO: return a pointer?
func NewSampler(config *vk.SamplerCreateInfo) ImageSampler {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	offset := samplerHeap.Alloc(vulkanSamplerDescriptorSize, &samplerHint)

	dst := samplerHeap.Base().Value()[offset:]
	vkFns.WriteSamplerDescriptorsEXT(device, 1, config, new(byteSliceToHostAddressRange(dst)))

	return ImageSampler{bits: uint32(offset/vulkanSamplerDescriptorSize) << 20}
}

// func (sampler ImageSampler) Descriptor() uint32 { return sampler.descriptor }

func (sampler ImageSampler) Destroy() {
	// TODO: check that the low bits are all zeros
	index := int(sampler.bits >> 20)
	if index == 0 {
		return
	}

	samplerHeap.Free(index*vulkanSamplerDescriptorSize, vulkanSamplerDescriptorSize)
}
