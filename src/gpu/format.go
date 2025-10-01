package gpu

import (
	"runtime"
	"sync"
	"unsafe"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

type Format = vk.Format

type IndexType = vk.IndexType

type formatProps struct {
	// TODO: replace with our own flags/bools?
	OptimalTilingFeatures vk.FormatFeatureFlags2
}

// TODO: when we get generic sync.Map, remove the indirection in front of values
var formatPropsCache sync.Map

func getFormatProps(format Format) formatProps {
	props, ok := formatPropsCache.Load(format)
	if !ok {
		props = getFormatPropsSlow(format)
		formatPropsCache.LoadOrStore(format, props)
	}
	return *props.(*formatProps)
}

func getFormatPropsSlow(format Format) *formatProps {
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

	return &formatProps{
		OptimalTilingFeatures: props3.OptimalTilingFeatures,
	}
}

func divByBlockExtent(texels [3]int, format Format) [3]int {
	blockExtent := int3FromVkExtent3D(formatutil.Describe(format).BlockExtent)

	// TODO: make sure texels is divisible by blockExtent?
	// TODO: we can speed the division up as there's only a small set of
	// different values that can appear in the sides of blockExtent, which we
	// can treat specially.

	return int3(texels).Div(blockExtent)
}

func divByBlockExtentRoundUp(texels [3]int, format Format) [3]int {
	blockExtent := int3FromVkExtent3D(formatutil.Describe(format).BlockExtent)

	return int3DivRoundUp(texels, blockExtent)
}
