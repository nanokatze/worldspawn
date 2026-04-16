package gpu

import (
	"unsafe"
	"worldspawn/gpu/vk"
)

var imageViews = make([]vk.ImageView, 1e6)

// TODO: rename to singular ImageDescriptor tbh
type ImageDescriptor struct {
	// TODO: explain how things are packed
	bits uint32
}

func newImageDescriptor(data *imageData, config *subImageConfig) ImageDescriptor {
	formatProps := getFormatImageProperties(config.Format)
	if !formatProps.Supported {
		panic("unsupported format")
	}

	usages := data.usages & formatProps.SupportedUsages

	// TODO: factor things into a mask of shader usages and then do a loop
	// creating a descriptor for every usage.
	sampling := usages&vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT) != 0
	loadStore := usages&vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT) != 0
	if !(sampling || loadStore) {
		return ImageDescriptor{}
	}

	index := 2 * resourceDescAlloc.Alloc(&resourceDescAllocHint)
	var tag uint32
	if sampling {
		initVulkanImageDescriptor(
			data,
			config,
			vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE,
			&imageViews[index+0],
			pointerToDescriptor{
				Set:          descriptorSet,
				Binding:      1,
				ArrayElement: uint32(index + 0),
			})
		tag |= 1 << 0
	}
	if loadStore {
		initVulkanImageDescriptor(
			data,
			config,
			vk.DESCRIPTOR_TYPE_STORAGE_IMAGE,
			&imageViews[index+1],
			pointerToDescriptor{
				Set:          descriptorSet,
				Binding:      2,
				ArrayElement: uint32(index + 1),
			})
		tag |= 1 << 1
	}

	return ImageDescriptor{uint32(index) | tag<<20}
}

func destroyImageDescriptor(descriptors ImageDescriptor) {
	index := int(descriptors.bits & (1<<20 - 1))
	tag := descriptors.bits >> 20

	if tag&(1<<0) != 0 {
		vkFns.DestroyImageView(device, imageViews[index+0], nil)
	}
	if tag&(1<<1) != 0 {
		vkFns.DestroyImageView(device, imageViews[index+1], nil)
	}
	resourceDescAlloc.Free(index / 2)
}

type pointerToDescriptor struct {
	Set          vk.DescriptorSet
	Binding      uint32
	ArrayElement uint32
}

func initVulkanImageDescriptor(data *imageData, config *subImageConfig, descriptorType vk.DescriptorType, imageView *vk.ImageView, descriptor pointerToDescriptor) {
	must(vkFns.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		PNext: unsafe.Pointer(&vk.ImageViewUsageCreateInfo{
			SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO,
			Usage: imageUsageFromDescriptorType(descriptorType),
		}),
		Image:            data.vkImage,
		ViewType:         config.Dim.vkImageViewType(),
		Format:           config.Format,
		SubresourceRange: config.bounds().VkImageSubresourceRange(config.Format),
	}, nil, imageView))

	vkFns.UpdateDescriptorSets(device,
		1, &vk.WriteDescriptorSet{
			SType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
			DstSet:          descriptor.Set,
			DstBinding:      descriptor.Binding,
			DstArrayElement: descriptor.ArrayElement,
			DescriptorCount: 1,
			DescriptorType:  descriptorType,
			PImageInfo: &vk.DescriptorImageInfo{
				ImageView:   *imageView,
				ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			},
		},
		0, nil)
}

func imageUsageFromDescriptorType(descriptorType vk.DescriptorType) vk.ImageUsageFlags {
	switch descriptorType {
	case vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE:
		return vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT)
	case vk.DESCRIPTOR_TYPE_STORAGE_IMAGE:
		return vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT)
	default:
		panic("unreachable")
	}
}
