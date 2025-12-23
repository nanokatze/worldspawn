package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

type imageFormatAllowedShaderUsages struct {
	Sampling  bool
	LoadStore bool
}

const (
	formatFeaturesRequiredForSampling = vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_SAMPLED_IMAGE_BIT)

	formatFeaturesRequiredForLoadStore = vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_IMAGE_BIT) |
		vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_READ_WITHOUT_FORMAT_BIT) |
		vk.FormatFeatureFlags2(vk.FORMAT_FEATURE_2_STORAGE_WRITE_WITHOUT_FORMAT_BIT)
)

func getImageFormatAllowedShaderUsages(format Format) imageFormatAllowedShaderUsages {
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

	return imageFormatAllowedShaderUsages{
		Sampling:  props3.OptimalTilingFeatures&formatFeaturesRequiredForSampling == formatFeaturesRequiredForSampling,
		LoadStore: props3.OptimalTilingFeatures&formatFeaturesRequiredForLoadStore == formatFeaturesRequiredForLoadStore,
	}
}
