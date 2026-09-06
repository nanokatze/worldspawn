package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: make this public?
const impliedUsage = 0 |
	vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_SRC_BIT) |
	vk.ImageUsageFlags(vk.IMAGE_USAGE_TRANSFER_DST_BIT)

type imageData struct {
	vkImage vk.Image

	config ImageConfig // TODO: use a more compact representation
	usage  vk.ImageUsageFlags

	memory *deviceMemory // TODO: replace with an UnsafePointer and length
}

func makeImageData(config ImageConfig, opts *newImageOptions) *imageData {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	data := &imageData{
		config: config,
		usage:  opts.usage,
	}

	switch {
	case opts.vkImage == vk.NULL_HANDLE:
		// TODO: don't poke insides of Config directly but use accessors instead?
		imageCreateInfo := new(vk.ImageCreateInfo)
		imageCreateInfo.SType = vk.STRUCTURE_TYPE_IMAGE_CREATE_INFO
		// TODO: shove these into a function that operates on vk.ImageCreateInfo
		imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_MUTABLE_FORMAT_BIT)
		imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_EXTENDED_USAGE_BIT)
		// TODO: actually check if it's compressed instead of checking BlockExtent
		if formatutil.Describe(config.format).BlockExtent != ([3]int{1, 1, 1}) {
			imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_BLOCK_TEXEL_VIEW_COMPATIBLE_BIT)
		}
		if config.dim.dimensions() == 2 && config.extent[0] == config.extent[1] && config.layers >= 6 {
			imageCreateInfo.Flags |= vk.ImageCreateFlags(vk.IMAGE_CREATE_CUBE_COMPATIBLE_BIT)
		}
		imageCreateInfo.ImageType = config.dim.vkImageType()
		imageCreateInfo.Format = config.format
		imageCreateInfo.Extent = vkExtent3DFromInt3(config.extent)
		imageCreateInfo.MipLevels = uint32(config.mips)
		imageCreateInfo.ArrayLayers = uint32(config.layers)
		imageCreateInfo.Samples = 1
		imageCreateInfo.Usage = opts.usage | impliedUsage
		imageCreateInfo.SharingMode = vk.SHARING_MODE_CONCURRENT
		imageCreateInfo.QueueFamilyIndexCount = uint32(len(Topology.Probe))
		imageCreateInfo.PQueueFamilyIndices = unsafe.SliceData(Topology.Probe)

		pinner.Pin(imageCreateInfo)
		pinner.Pin(imageCreateInfo.PQueueFamilyIndices)

		requirements := &vk.MemoryRequirements2{
			SType: vk.STRUCTURE_TYPE_MEMORY_REQUIREMENTS_2,
		}
		VkFns.GetDeviceImageMemoryRequirements(Device,
			&vk.DeviceImageMemoryRequirements{
				SType:       vk.STRUCTURE_TYPE_DEVICE_IMAGE_MEMORY_REQUIREMENTS,
				PCreateInfo: imageCreateInfo,
			},
			requirements)

		size := roundUpDeviceAllocationSize(int(requirements.Size))

		var vkImage vk.Image
		must(VkFns.CreateImage(Device, imageCreateInfo, nil, &vkImage))

		memoryTypeIndex := findMemoryTypeIndex(requirements.MemoryTypeBits, 0)

		var memory *deviceMemory
		{
			allocPoolMu.Lock()
			entries := allocPool[size]
			if len(entries) > 0 {
				memory = entries[len(entries)-1]
				allocPool[size] = entries[:len(entries)-1]
			}
			allocPoolMu.Unlock()
		}
		if memory == nil {
			var allocation deviceMemory
			allocation.size = size
			must(VkFns.AllocateMemory(Device, &vk.MemoryAllocateInfo{
				SType:           vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
				AllocationSize:  vk.DeviceSize(size),
				MemoryTypeIndex: uint32(memoryTypeIndex),
			}, nil, &allocation.memory))
			memory = &allocation
		}

		must(VkFns.BindImageMemory2(Device, 1, &vk.BindImageMemoryInfo{
			SType:        vk.STRUCTURE_TYPE_BIND_IMAGE_MEMORY_INFO,
			Image:        vkImage,
			Memory:       memory.memory,
			MemoryOffset: 0,
		}))

		data.vkImage = vkImage
		data.memory = memory

	case opts.vkImage != vk.NULL_HANDLE:
		data.vkImage = opts.vkImage
	}

	return data
}

func (data *imageData) destroy() {
	VkFns.DestroyImage(Device, data.vkImage, nil)

	allocPoolMu.Lock()
	allocPool[data.memory.size] = append(allocPool[data.memory.size], data.memory)
	allocPoolMu.Unlock()
}
