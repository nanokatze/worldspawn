package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type formatImageProperties struct {
	Supported       bool
	SupportedUsages vk.ImageUsageFlags
}

var getFormatImageProperties = cached(func(format vk.Format) formatImageProperties {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	props3 := &vk.FormatProperties3{
		SType: vk.STRUCTURE_TYPE_FORMAT_PROPERTIES_3,
	}
	pinner.Pin(props3)

	vkFns.GetPhysicalDeviceFormatProperties2(physicalDevice,
		format,
		&vk.FormatProperties2{
			SType: vk.STRUCTURE_TYPE_FORMAT_PROPERTIES_2,
			PNext: unsafe.Pointer(props3),
		})

	if !testAllSet(props3.OptimalTilingFeatures, vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_TRANSFER_SRC_BIT)|vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_TRANSFER_DST_BIT)) {
		return formatImageProperties{}
	}

	var supportedUsages vk.ImageUsageFlags
	if testAllSet(props3.OptimalTilingFeatures, vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_SAMPLED_IMAGE_BIT)) {
		supportedUsages |= vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT)
	}
	if testAllSet(props3.OptimalTilingFeatures, vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_IMAGE_BIT)) {
		supportedUsages |= vk.ImageUsageFlags(vk.IMAGE_USAGE_STORAGE_BIT)
	}

	return formatImageProperties{
		Supported:       true,
		SupportedUsages: supportedUsages,
	}
})

// TODO: rename this to something else
func testAllSet[T ~uint64](x, y T) bool { return x&y == y }
