package gpu

import (
	"math/bits"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

// Filled at init
//
// TODO: move this closer to where it's inited once we move this whole pile of
// code into image/
var (
	vulkanImageDescriptorSize   int
	vulkanSamplerDescriptorSize int
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

	offset := resourceHeap.Alloc(bits.Len32(tag)*vulkanImageDescriptorSize, &imageHint)

	dst0 := resourceHeap.Base().Value()[offset:]
	for i := range ones32(tag) {
		dst := dst0[i*vulkanImageDescriptorSize:]
		switch i {
		case 0:
			initVulkanImageDescriptor(dst, data, config, vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE)
		case 1:
			initVulkanImageDescriptor(dst, data, config, vk.DESCRIPTOR_TYPE_STORAGE_IMAGE)
		default:
			panic("unreachable")
		}
	}

	return ImageDescriptor{uint32(offset/vulkanImageDescriptorSize) | tag<<20}
}

func destroyImageDescriptor(descriptor ImageDescriptor) {
	off := int(descriptor.bits&(1<<20-1)) * vulkanImageDescriptorSize

	tag := descriptor.bits >> 20
	len := bits.Len32(tag) * vulkanImageDescriptorSize

	resourceHeap.Free(off, len)
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
