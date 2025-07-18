package gpu

import (
	"fmt"
	"runtime"

	"worldspawn/gpu/vk"
)

// TODO: move this file into image.go tbh

type Sampler struct{ handle uint32 }

var samplerAllocHint int64
var samplerSlots = newSlotAlloc(2e3)
var samplerObjects = make([]vk.Sampler, 2e3)

func NewSampler(config *vk.SamplerCreateInfo) Sampler {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	index := samplerSlots.Alloc(&samplerAllocHint)

	var vkSampler vk.Sampler
	if err := vkFns.CreateSampler(device, config, nil, &vkSampler); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateSampler: %v", err))
	}

	vkFns.UpdateDescriptorSets(device,
		1, &vk.WriteDescriptorSet{
			SType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
			DstSet:          descriptorSet,
			DstBinding:      0,
			DstArrayElement: uint32(index),
			DescriptorCount: 1,
			DescriptorType:  vk.DESCRIPTOR_TYPE_SAMPLER,
			PImageInfo:      pinned(&pinner, &vk.DescriptorImageInfo{Sampler: vkSampler}),
		},
		0, nil)
	samplerObjects[index] = vkSampler

	return Sampler{handle: uint32(index) << 20}
}

func (sampler Sampler) Destroy() {
	// TODO: check that the low bits are all zeros
	index := int(sampler.handle >> 20)
	if index == 0 {
		return
	}

	// TODO: do extra checks here

	vkFns.DestroySampler(device, samplerObjects[index], nil)

	samplerObjects[index] = vk.NULL_HANDLE
	samplerSlots.Free(index) // must be done last
}
