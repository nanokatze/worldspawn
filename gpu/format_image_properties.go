package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type formatImageProperties struct {
	Supported      bool
	SupportedUsage vk.ImageUsageFlags
}

var getFormatImageProperties = cached(func(format vk.Format) formatImageProperties {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	props3 := &vk.FormatProperties3{
		SType: vk.STRUCTURE_TYPE_FORMAT_PROPERTIES_3,
	}
	pinner.Pin(props3)

	VkFns.GetPhysicalDeviceFormatProperties2(PhysicalDevice,
		format,
		&vk.FormatProperties2{
			SType: vk.STRUCTURE_TYPE_FORMAT_PROPERTIES_2,
			PNext: unsafe.Pointer(props3),
		})

	const requiredFeatures = 0 |
		vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_TRANSFER_SRC_BIT) |
		vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_TRANSFER_DST_BIT)

	supported := props3.OptimalTilingFeatures&requiredFeatures == requiredFeatures

	if !supported {
		return formatImageProperties{}
	}

	var supportedUsage vk.ImageUsageFlags
	if props3.OptimalTilingFeatures&vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_SAMPLED_IMAGE_BIT) != 0 {
		supportedUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT)
	}
	if props3.OptimalTilingFeatures&vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_IMAGE_BIT) != 0 {
		supportedUsage |= vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT)
	}

	return formatImageProperties{
		Supported:      true,
		SupportedUsage: supportedUsage,
	}
})
