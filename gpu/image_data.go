package gpu

import (
	"worldspawn/gpu/vk"
)

type imageData struct {
	vkImage vk.Image

	dim    int
	format vk.Format
	extent [3]int
	layers int
	mips   int
	usages vk.ImageUsageFlags // we don't need to store all usages, just sampling and storage

	memory *deviceMemory // TODO: replace with an UnsafePointer and length
}

func (data *imageData) destroy() {
	vkFns.DestroyImage(device, data.vkImage, nil)

	allocPoolMu.Lock()
	allocPool[data.memory.size] = append(allocPool[data.memory.size], data.memory)
	allocPoolMu.Unlock()
}
