package gpu

import (
	"math/bits"

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

// type shaderUsage uint8

const (
	shaderUsageSampling = 1 << iota
	shaderUsageLoadStore
)

type ImageDescriptor struct{ bits uint32 }

func packImageDescriptor(unscaledOffset int, tag uint32) ImageDescriptor {
	// TODO: validate unscaledOffset and tag
	return ImageDescriptor{
		bits: uint32(unscaledOffset) | tag<<20,
	}
}

func (descriptor ImageDescriptor) unscaledOffsetBase() int {
	return int(descriptor.bits & ((1 << 20) - 1))
}

func (descriptor ImageDescriptor) tag() uint32 { return descriptor.bits >> 20 }

// TODO: introduce a function of tag for computing the length of the would-be
// ImageDescriptor with the said tag. Keep it host-only in case we ever need to
// read device properties to determine the length of some compound descriptors.
// The compound descriptors would always have to be at the end.

// TODO: the hints should be per-thread or similar.
// TODO: keep separate hints for 1 and 2 -descriptor allocations?

var imageDescriptorAllocHint int64

func newImageDescriptor(data *imageData, config *subImageConfig) ImageDescriptor {
	formatProps := getFormatImageProperties(config.format)
	if !formatProps.Supported {
		panic("unsupported format")
	}

	usage := data.usage & formatProps.SupportedUsage

	var tag uint32
	if usage&vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT) != 0 {
		tag |= shaderUsageSampling
	}
	if usage&vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT) != 0 {
		tag |= shaderUsageLoadStore
	}

	if tag == 0 {
		return ImageDescriptor{}
	}

	offset := resourceHeap.Alloc(bits.OnesCount32(tag)*vulkanImageDescriptorSize, &imageDescriptorAllocHint)

	dst0 := resourceHeap.Base().Value()[offset:]
	for i := range ones32(tag) {
		dst := dst0[rank32(tag, i)*vulkanImageDescriptorSize:]
		switch 1 << i {
		case shaderUsageSampling:
			marshalVulkanImageDescriptor(dst, data, config, vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE)
		case shaderUsageLoadStore:
			marshalVulkanImageDescriptor(dst, data, config, vk.DESCRIPTOR_TYPE_STORAGE_IMAGE)
		default:
			panic("unreachable")
		}
	}

	return packImageDescriptor(offset/vulkanImageDescriptorSize, tag)
}

func cleanupImageDescriptor(descriptor ImageDescriptor) {
	off := descriptor.unscaledOffsetBase() * vulkanImageDescriptorSize
	len := bits.OnesCount32(descriptor.tag()) * vulkanImageDescriptorSize

	resourceHeap.Free(off, len)
}

func marshalVulkanImageDescriptor(dst []byte, data *imageData, config *subImageConfig, descriptorType vk.DescriptorType) {
	dstHostAddressRange := byteSliceToHostAddressRange(dst)

	VkFns.WriteResourceDescriptorsEXT(Device, 1,
		&vk.ResourceDescriptorInfoEXT{
			SType: vk.STRUCTURE_TYPE_RESOURCE_DESCRIPTOR_INFO_EXT,
			Type:  descriptorType,
			Data: vk.ResourceDescriptorDataEXT(&vk.ImageDescriptorInfoEXT{
				SType: vk.STRUCTURE_TYPE_IMAGE_DESCRIPTOR_INFO_EXT,
				PView: &vk.ImageViewCreateInfo{
					SType:            vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
					Image:            data.vkImage,
					ViewType:         config.dim.vkImageViewType(),
					Format:           config.format,
					SubresourceRange: config.bounds().VkImageSubresourceRange(config.format),
				},
				Layout: vk.IMAGE_LAYOUT_GENERAL,
			}),
		},
		&dstHostAddressRange)
}

func rank32(x uint32, i int) int { return bits.OnesCount32(x & ((1 << i) - 1)) }
