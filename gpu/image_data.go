package gpu

import (
	"unsafe"

	"worldspawn/gpu/vk"
)

type imageData struct {
	vkImage vk.Image

	// TODO: review which of these fields we need
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

type pointerToDescriptor struct {
	Set          vk.DescriptorSet
	Binding      uint32
	ArrayElement uint32
}

// TODO: drop the use of subImageConfig use here?
func (data *imageData) shaderDescriptor(config *subImageConfig, descriptorType vk.DescriptorType, imageView *vk.ImageView, descriptor pointerToDescriptor) {
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
