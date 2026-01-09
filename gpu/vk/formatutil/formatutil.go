//go:generate go run gen.go -o format_table.go /usr/share/vulkan/registry/vk.xml
//go:generate stringer -type Class -trimprefix CLASS_

package formatutil

import "worldspawn/gpu/vk"

type Class int

type Description struct {
	Class       Class
	BlockSize   uint32
	BlockExtent vk.Extent3D
}

func Describe(format vk.Format) Description {
	return formatTable[format]
}

func FindFormat(description Description) vk.Format {
	return 0
}

// TODO: derive this from the format table and/or autogenerate?
// TODO: eventually get rid of this and push constructing image aspect flags
// onto the boundary code interacting with vk.
func Aspects(format vk.Format) vk.ImageAspectFlags {
	switch format {
	default:
		// This is incorrect for planar formats, but we don't care about those
		// for now.
		return vk.ImageAspectFlags(vk.IMAGE_ASPECT_COLOR_BIT)
	case vk.FORMAT_D16_UNORM, vk.FORMAT_X8_D24_UNORM_PACK32, vk.FORMAT_D32_SFLOAT:
		return vk.ImageAspectFlags(vk.IMAGE_ASPECT_DEPTH_BIT)
	case vk.FORMAT_S8_UINT:
		return vk.ImageAspectFlags(vk.IMAGE_ASPECT_STENCIL_BIT)
	case vk.FORMAT_D16_UNORM_S8_UINT, vk.FORMAT_D24_UNORM_S8_UINT, vk.FORMAT_D32_SFLOAT_S8_UINT:
		return vk.ImageAspectFlags(vk.IMAGE_ASPECT_DEPTH_BIT) | vk.ImageAspectFlags(vk.IMAGE_ASPECT_STENCIL_BIT)
	}
}
