package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type ImageDescriptor struct {
	bits uint32
}

// TODO: add (*Image).Supports(usage)

func newImageDescriptor(data *imageData, config subImageConfig) ImageDescriptor {
	formatProps := getFormatImageProperties(config.format)
	if !formatProps.Supported {
		panic("unsupported format")
	}

	usage := data.usage & formatProps.SupportedUsage

	// TODO: factor things into a mask of shader usage bits and then do a loop
	// creating a descriptor for every usage.
	sampling := usage&vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT) != 0
	loadStore := usage&vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT) != 0
	if !(sampling || loadStore) {
		return ImageDescriptor{}
	}

	rd := ResourceDescriptor(resourceHeap.Alloc(&resourceDescAllocHint))
	rdMap := rd.Map().Value()
	var tag uint32
	if sampling {
		initVulkanImageDescriptor(rdMap, data, config, vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE)
		tag |= 1 << 0
	}
	if loadStore {
		initVulkanImageDescriptor(rdMap[resourceHeap.elemSize/2:], data, config, vk.DESCRIPTOR_TYPE_STORAGE_IMAGE)
		tag |= 1 << 1
	}

	return ImageDescriptor{uint32(2*rd) | tag<<20}
}

func destroyImageDescriptor(descriptor ImageDescriptor) {
	index := int(descriptor.bits & (1<<20 - 1))
	tag := descriptor.bits >> 20

	if tag&(1<<0) != 0 {
	}
	if tag&(1<<1) != 0 {
	}
	resourceHeap.Free(index / 2)
}

func initVulkanImageDescriptor(dst []byte, data *imageData, config subImageConfig, descriptorType vk.DescriptorType) {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	pinner.Pin(unsafe.SliceData(dst))

	vkFns.WriteResourceDescriptorsEXT(device, 1,
		&vk.ResourceDescriptorInfoEXT{
			SType: vk.STRUCTURE_TYPE_RESOURCE_DESCRIPTOR_INFO_EXT,
			Type:  descriptorType,
			Data: vk.ResourceDescriptorDataEXT(pinned(&pinner, &vk.ImageDescriptorInfoEXT{
				SType: vk.STRUCTURE_TYPE_IMAGE_DESCRIPTOR_INFO_EXT,
				PView: pinned(&pinner, &vk.ImageViewCreateInfo{
					SType:            vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
					Image:            data.vkImage,
					ViewType:         config.dim.vkImageViewType(),
					Format:           config.format,
					SubresourceRange: config.bounds().VkImageSubresourceRange(config.format),
				}),
				Layout: vk.IMAGE_LAYOUT_GENERAL,
			})),
		},
		new(byteSliceToHostAddressRange(dst)))
}
