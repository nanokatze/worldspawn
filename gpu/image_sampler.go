package gpu

import (
	"runtime"

	"worldspawn/gpu/vk"
)

type ImageSampler struct{ descriptor uint32 }

var samplerHint int64

// TODO: return a pointer?
func NewSampler(config *vk.SamplerCreateInfo) ImageSampler {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	heap := &samplerHeap

	index := heap.Alloc(1, &samplerHint)

	dst := heap.Map(index)

	vkFns.WriteSamplerDescriptorsEXT(device, 1, config, new(byteSliceToHostAddressRange(dst.Value())))

	return ImageSampler{descriptor: uint32(index) << 20}
}

// func (sampler ImageSampler) Descriptor() uint32 { return sampler.descriptor }

func (sampler ImageSampler) Destroy() {
	// TODO: check that the low bits are all zeros
	index := int(sampler.descriptor >> 20)
	if index == 0 {
		return
	}

	samplerHeap.Free(index)
}
