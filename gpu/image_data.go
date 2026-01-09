package gpu

import (
	"worldspawn/gpu/vk"
)

type imageData struct {
	vkImage vk.Image

	// TODO: review which of these fields we need
	dim       ImageDim
	extent    vk.Extent3D
	layers    uint32
	mipLevels uint32
	format    Format
	usage     ImageUsage

	memory *deviceMemory // TODO: replace with an UnsafePointer and length
}

func (base *imageData) destroy() {
	vkFns.DestroyImage(device, base.vkImage, nil)

	allocPoolMu.Lock()
	allocPool[base.memory.size] = append(allocPool[base.memory.size], base.memory)
	allocPoolMu.Unlock()
}
