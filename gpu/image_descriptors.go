package gpu

import (
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

var imageViews = make([]vk.ImageView, 1e6)

type imageDescriptors struct {
	sampling, loadStore uint32
}

func newImageDescriptors(data *imageData, config *subImageConfig) imageDescriptors {
	formatProps := getFormatImageProperties(config.Format)
	if !formatProps.Supported {
		panic("unsupported format")
	}

	var descriptors imageDescriptors
	usages := data.usages & formatProps.SupportedUsages
	if usages&vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT) != 0 {
		descriptors.sampling = newImageDescriptor(data, config, vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE)
	}
	if usages&vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT) != 0 {
		descriptors.loadStore = newImageDescriptor(data, config, vk.DESCRIPTOR_TYPE_STORAGE_IMAGE)
	}
	return descriptors
}

func newImageDescriptor(base *imageData, config *subImageConfig, descriptorType vk.DescriptorType) uint32 {
	dstBinding := uint32(1)
	if descriptorType == vk.DESCRIPTOR_TYPE_STORAGE_IMAGE {
		dstBinding = 2
	}
	descriptor := uint32(resourceDescAlloc.Alloc(&resourceDescAllocHint))
	base.shaderDescriptor(
		config,
		descriptorType,
		&imageViews[descriptor],
		pointerToDescriptor{
			Set:          descriptorSet,
			Binding:      dstBinding,
			ArrayElement: uint32(descriptor),
		})
	return descriptor
}

func (descriptors imageDescriptors) destroy() {
	vkFns.DestroyImageView(device, imageViews[descriptors.sampling], nil)
	resourceDescAlloc.Free(int(descriptors.sampling))
	vkFns.DestroyImageView(device, imageViews[descriptors.loadStore], nil)
	resourceDescAlloc.Free(int(descriptors.loadStore))
}

type pointerToDescriptor struct {
	Set          vk.DescriptorSet
	Binding      uint32
	ArrayElement uint32
}

func (data *imageData) shaderDescriptor(config *subImageConfig, descriptorType vk.DescriptorType, imageView *vk.ImageView, descriptor pointerToDescriptor) {
	must(vkFns.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		PNext: unsafe.Pointer(&vk.ImageViewUsageCreateInfo{
			SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO,
			Usage: imageUsageFromDescriptorType(descriptorType),
		}),
		Image:    data.vkImage,
		ViewType: config.Dim.vkImageViewType(),
		Format:   config.Format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     formatutil.Aspects(config.Format),
			BaseMipLevel:   uint32(config.FirstMip),
			LevelCount:     uint32(config.Mips),
			BaseArrayLayer: uint32(config.FirstLayer),
			LayerCount:     uint32(config.Layers),
		},
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
