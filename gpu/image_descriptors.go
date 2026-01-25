package gpu

import "worldspawn/gpu/vk"

var imageViews = make([]vk.ImageView, 1e6)

type ImageDescriptors struct {
	// TODO: explain how things are packed
	bits uint32
}

func newImageDescriptors(data *imageData, config *subImageConfig) ImageDescriptors {
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
		return ImageDescriptors{}
	}

	index := 2 * resourceDescAlloc.Alloc(&resourceDescAllocHint)
	var tag uint32
	if sampling {
		data.shaderDescriptor(
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
		data.shaderDescriptor(
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

	return ImageDescriptors{uint32(index) | tag<<20}
}

func destroyImageDescriptors(descriptors ImageDescriptors) {
	index := int(descriptors.bits & (1<<20 - 1))

	if descriptors.bits&(1<<20) != 0 {
		vkFns.DestroyImageView(device, imageViews[index+0], nil)
	}
	if descriptors.bits&(1<<21) != 0 {
		vkFns.DestroyImageView(device, imageViews[index+1], nil)
	}
	resourceDescAlloc.Free(index / 2)
}
