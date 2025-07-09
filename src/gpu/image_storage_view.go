package gpu

import (
	"fmt"
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type StorageView struct{ handle uint32 }

func newStorageView(image *Image) StorageView {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	index := imageDescriptors.Alloc(&imageDescAllocHint)

	var vkView vk.ImageView
	if err := vkFns.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		PNext: unsafe.Pointer(pinned(&pinner, &vk.ImageViewUsageCreateInfo{
			SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO,
			Usage: vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT),
		})),
		Image:            image.base.vkImage,
		ViewType:         image.Dim().vkImageViewType(),
		Format:           image.Format(),
		SubresourceRange: vkImageSubresourceRange(image),
	}, nil, &vkView); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateImageView: %v", err))
	}

	vkFns.UpdateDescriptorSets(device,
		1, &vk.WriteDescriptorSet{
			SType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
			DstSet:          descriptorSet,
			DstBinding:      2,
			DstArrayElement: uint32(index),
			DescriptorCount: 1,
			DescriptorType:  vk.DESCRIPTOR_TYPE_STORAGE_IMAGE,
			PImageInfo: pinned(&pinner, &vk.DescriptorImageInfo{
				ImageView:   vkView,
				ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			}),
		},
		0, nil)
	imageViews[index] = uint64(vkView)

	// NOTE: because we don't use the entire 4096 samplers, so we could set the
	// 20:12 to all ones in storage views.
	return StorageView{handle: uint32(index)}
}

func (storageView StorageView) Destroy() {
	index := int(storageView.handle)
	if index == 0 {
		return
	}

	// TODO: do extra checks here

	vkFns.DestroyImageView(device, vk.ImageView(imageViews[index]), nil)

	// TODO: consider changing stuff here so as to "quarantine" handles for a while

	imageViews[index] = vk.NULL_HANDLE
	imageDescriptors.Free(index) // must be done last
}
