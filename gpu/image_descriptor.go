package gpu

import (
	"math/bits"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type ImageDescriptor struct {
	bits uint32
}

// TODO: add (*Image).Supports(usage)

// TODO: the hints should be per-thread or similar.
// TODO: keep separate hints for 1 and 2 -descriptor allocations?

var imageHint int64

func newImageDescriptor(data *imageData, config subImageConfig) ImageDescriptor {
	formatProps := getFormatImageProperties(config.format)
	if !formatProps.Supported {
		panic("unsupported format")
	}

	usage := data.usage & formatProps.SupportedUsage

	var tag uint32
	if usage&vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT) != 0 {
		tag |= 1 << 0
	}
	if usage&vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT) != 0 {
		tag |= 1 << 1
	}

	if tag == 0 {
		return ImageDescriptor{}
	}

	// TODO: switch to bits.OnesCount32(tag) eventually

	rd := ResourceDescriptor(resourceHeap.Alloc(bits.Len32(tag), &imageHint))
	for i := range ones32(tag) {
		dst := (rd + ResourceDescriptor(i)).Map().Value()
		switch i {
		case 0:
			initVulkanImageDescriptor(dst, data, config, vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE)
		case 1:
			initVulkanImageDescriptor(dst, data, config, vk.DESCRIPTOR_TYPE_STORAGE_IMAGE)
		default:
			panic("unreachable")
		}
	}

	return ImageDescriptor{uint32(rd) | tag<<20}
}

func destroyImageDescriptor(descriptor ImageDescriptor) {
	index := int(descriptor.bits & (1<<20 - 1))
	tag := descriptor.bits >> 20

	for i := range bits.Len32(tag) {
		resourceHeap.Free(index + i)
	}
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
