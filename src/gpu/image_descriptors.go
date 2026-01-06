package gpu

import (
	"worldspawn/gpu/vk"
)

// TODO: rename these

var imageDescAlloc = newSlotAlloc(1e6) // allocate either at NewImage() or gpuInit()
var imageDescAllocHint int64           // can we m ake this per-thread?
var imageViews = make([]vk.ImageView, 1e6)

type imageDescriptors struct {
	sampling, loadStore uint32
}

func newImageDescriptors(
	base *imageData,
	dim ImageDim,
	format Format,
	baseLayer, layers int,
	baseMipLevel, mipLevels int) imageDescriptors {
	formatProps := getFormatImageProperties(format)
	if !formatProps.Supported {
		panic("unsupported format")
	}

	usage := base.usage & formatProps.SupportedUsages

	var descriptors imageDescriptors
	if usage&ImageUsageSampling != 0 {
		descriptors.sampling = newImageDescriptor(
			base,
			dim,
			format,
			baseLayer, layers,
			baseMipLevel, mipLevels,
			vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE)
	}
	if usage&ImageUsageLoadStore != 0 {
		descriptors.loadStore = newImageDescriptor(
			base,
			dim,
			format,
			baseLayer, layers,
			baseMipLevel, mipLevels,
			vk.DESCRIPTOR_TYPE_STORAGE_IMAGE)
	}
	return descriptors
}

func newImageDescriptor(
	base *imageData,
	dim ImageDim,
	format Format,
	baseLayer, layers int,
	baseMipLevel, mipLevels int,
	descriptorType vk.DescriptorType) uint32 {
	dstBinding := uint32(1)
	if descriptorType == vk.DESCRIPTOR_TYPE_STORAGE_IMAGE {
		dstBinding = 2
	}
	descriptor := uint32(imageDescAlloc.Alloc(&imageDescAllocHint))
	base.getShaderDescriptor(
		dim,
		format,
		baseLayer, layers,
		baseMipLevel, mipLevels,
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
	imageDescAlloc.Free(int(descriptors.sampling))
	vkFns.DestroyImageView(device, imageViews[descriptors.loadStore], nil)
	imageDescAlloc.Free(int(descriptors.loadStore))
}
