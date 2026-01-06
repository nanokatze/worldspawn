package gpu

import (
	"unsafe"
	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
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

type pointerToDescriptor struct {
	Set          vk.DescriptorSet
	Binding      uint32
	ArrayElement uint32
}

func (base *imageData) getShaderDescriptor(
	dim ImageDim,
	format Format,
	baseLayer, layers int,
	baseMipLevel, mipLevels int,
	descriptorType vk.DescriptorType,
	outImageView *vk.ImageView,
	outDescriptor pointerToDescriptor) {
	must(vkFns.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		PNext: unsafe.Pointer(&vk.ImageViewUsageCreateInfo{
			SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO,
			Usage: imageUsageFromDescriptorType(descriptorType),
		}),
		Image:    base.vkImage,
		ViewType: dim.vkImageViewType(),
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     formatutil.Aspects(format),
			BaseMipLevel:   uint32(baseMipLevel),
			LevelCount:     uint32(mipLevels),
			BaseArrayLayer: uint32(baseLayer),
			LayerCount:     uint32(layers),
		},
	}, nil, outImageView))

	vkFns.UpdateDescriptorSets(device,
		1, &vk.WriteDescriptorSet{
			SType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
			DstSet:          outDescriptor.Set,
			DstBinding:      outDescriptor.Binding,
			DstArrayElement: outDescriptor.ArrayElement,
			DescriptorCount: 1,
			DescriptorType:  descriptorType,
			PImageInfo: &vk.DescriptorImageInfo{
				ImageView:   *outImageView,
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
