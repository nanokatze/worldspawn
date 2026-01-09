package formatutil

import "worldspawn/gpu/vk"

const (
	_ Class = iota
	CLASS_10BIT_2PLANE_420
	CLASS_10BIT_2PLANE_422
	CLASS_10BIT_2PLANE_444
	CLASS_10BIT_3PLANE_420
	CLASS_10BIT_3PLANE_422
	CLASS_10BIT_3PLANE_444
	CLASS_12BIT_2PLANE_420
	CLASS_12BIT_2PLANE_422
	CLASS_12BIT_2PLANE_444
	CLASS_12BIT_3PLANE_420
	CLASS_12BIT_3PLANE_422
	CLASS_12BIT_3PLANE_444
	CLASS_128BIT
	CLASS_14BIT_2PLANE_420
	CLASS_14BIT_2PLANE_422
	CLASS_16BIT
	CLASS_16BIT_2PLANE_420
	CLASS_16BIT_2PLANE_422
	CLASS_16BIT_2PLANE_444
	CLASS_16BIT_3PLANE_420
	CLASS_16BIT_3PLANE_422
	CLASS_16BIT_3PLANE_444
	CLASS_192BIT
	CLASS_24BIT
	CLASS_256BIT
	CLASS_32BIT
	CLASS_32BIT_B8G8R8G8
	CLASS_32BIT_G8B8G8R8
	CLASS_48BIT
	CLASS_64BIT
	CLASS_64BIT_B10G10R10G10
	CLASS_64BIT_B12G12R12G12
	CLASS_64BIT_B16G16R16G16
	CLASS_64BIT_G10B10G10R10
	CLASS_64BIT_G12B12G12R12
	CLASS_64BIT_G16B16G16R16
	CLASS_64BIT_R10G10B10A10
	CLASS_64BIT_R12G12B12A12
	CLASS_64BIT_R14G14B14A14
	CLASS_8BIT
	CLASS_8BIT_2PLANE_420
	CLASS_8BIT_2PLANE_422
	CLASS_8BIT_2PLANE_444
	CLASS_8BIT_3PLANE_420
	CLASS_8BIT_3PLANE_422
	CLASS_8BIT_3PLANE_444
	CLASS_8BIT_ALPHA
	CLASS_96BIT
	CLASS_ASTC_10X10
	CLASS_ASTC_10X5
	CLASS_ASTC_10X6
	CLASS_ASTC_10X8
	CLASS_ASTC_12X10
	CLASS_ASTC_12X12
	CLASS_ASTC_4X4
	CLASS_ASTC_5X4
	CLASS_ASTC_5X5
	CLASS_ASTC_6X5
	CLASS_ASTC_6X6
	CLASS_ASTC_8X5
	CLASS_ASTC_8X6
	CLASS_ASTC_8X8
	CLASS_BC1_RGB
	CLASS_BC1_RGBA
	CLASS_BC2
	CLASS_BC3
	CLASS_BC4
	CLASS_BC5
	CLASS_BC6H
	CLASS_BC7
	CLASS_D16
	CLASS_D16S8
	CLASS_D24
	CLASS_D24S8
	CLASS_D32
	CLASS_D32S8
	CLASS_EAC_R
	CLASS_EAC_RG
	CLASS_ETC2_EAC_RGBA
	CLASS_ETC2_RGB
	CLASS_ETC2_RGBA
	CLASS_PVRTC1_2BPP
	CLASS_PVRTC1_4BPP
	CLASS_PVRTC2_2BPP
	CLASS_PVRTC2_4BPP
	CLASS_S8
)

var formatTable = map[vk.Format]Description{
	vk.FORMAT_R4G4_UNORM_PACK8: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R4G4B4A4_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B4G4R4A4_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R5G6B5_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B5G6R5_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R5G5B5A1_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B5G5R5A1_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A1R5G5B5_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A1B5G5R5_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8_UNORM: {
		Class:       CLASS_8BIT_ALPHA,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_UNORM: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_SNORM: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_USCALED: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_SSCALED: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_UINT: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_SINT: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_SRGB: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_UNORM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_SNORM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_USCALED: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_SSCALED: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_UINT: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_SINT: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8_SRGB: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_UNORM: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_SNORM: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_USCALED: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_SSCALED: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_UINT: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_SINT: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8_SRGB: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_UNORM: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_SNORM: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_USCALED: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_SSCALED: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_UINT: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_SINT: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8_SRGB: {
		Class:       CLASS_24BIT,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_UNORM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_SNORM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_USCALED: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_SSCALED: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_UINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_SINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8G8B8A8_SRGB: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_UNORM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_SNORM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_USCALED: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_SSCALED: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_UINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_SINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8A8_SRGB: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_UNORM_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_SNORM_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_USCALED_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_SSCALED_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_UINT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_SINT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A8B8G8R8_SRGB_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2R10G10B10_UNORM_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2R10G10B10_SNORM_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2R10G10B10_USCALED_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2R10G10B10_SSCALED_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2R10G10B10_UINT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2R10G10B10_SINT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2B10G10R10_UNORM_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2B10G10R10_SNORM_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2B10G10R10_USCALED_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2B10G10R10_SSCALED_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2B10G10R10_UINT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A2B10G10R10_SINT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_UNORM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_SNORM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_USCALED: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_SSCALED: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_UINT: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_SINT: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16_SFLOAT: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_UNORM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_SNORM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_USCALED: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_SSCALED: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_UINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_SINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_SFLOAT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_UNORM: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_SNORM: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_USCALED: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_SSCALED: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_UINT: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_SINT: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16_SFLOAT: {
		Class:       CLASS_48BIT,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_UNORM: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_SNORM: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_USCALED: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_SSCALED: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_UINT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_SINT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16B16A16_SFLOAT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32_UINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32_SINT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32_SFLOAT: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32_UINT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32_SINT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32_SFLOAT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32B32_UINT: {
		Class:       CLASS_96BIT,
		BlockSize:   12,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32B32_SINT: {
		Class:       CLASS_96BIT,
		BlockSize:   12,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32B32_SFLOAT: {
		Class:       CLASS_96BIT,
		BlockSize:   12,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32B32A32_UINT: {
		Class:       CLASS_128BIT,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32B32A32_SINT: {
		Class:       CLASS_128BIT,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R32G32B32A32_SFLOAT: {
		Class:       CLASS_128BIT,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64_UINT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64_SINT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64_SFLOAT: {
		Class:       CLASS_64BIT,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64_UINT: {
		Class:       CLASS_128BIT,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64_SINT: {
		Class:       CLASS_128BIT,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64_SFLOAT: {
		Class:       CLASS_128BIT,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64B64_UINT: {
		Class:       CLASS_192BIT,
		BlockSize:   24,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64B64_SINT: {
		Class:       CLASS_192BIT,
		BlockSize:   24,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64B64_SFLOAT: {
		Class:       CLASS_192BIT,
		BlockSize:   24,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64B64A64_UINT: {
		Class:       CLASS_256BIT,
		BlockSize:   32,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64B64A64_SINT: {
		Class:       CLASS_256BIT,
		BlockSize:   32,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R64G64B64A64_SFLOAT: {
		Class:       CLASS_256BIT,
		BlockSize:   32,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_B10G11R11_UFLOAT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_E5B9G9R9_UFLOAT_PACK32: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_D16_UNORM: {
		Class:       CLASS_D16,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_X8_D24_UNORM_PACK32: {
		Class:       CLASS_D24,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_D32_SFLOAT: {
		Class:       CLASS_D32,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_S8_UINT: {
		Class:       CLASS_S8,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_D16_UNORM_S8_UINT: {
		Class:       CLASS_D16S8,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_D24_UNORM_S8_UINT: {
		Class:       CLASS_D24S8,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_D32_SFLOAT_S8_UINT: {
		Class:       CLASS_D32S8,
		BlockSize:   5,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_BC1_RGB_UNORM_BLOCK: {
		Class:       CLASS_BC1_RGB,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC1_RGB_SRGB_BLOCK: {
		Class:       CLASS_BC1_RGB,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC1_RGBA_UNORM_BLOCK: {
		Class:       CLASS_BC1_RGBA,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC1_RGBA_SRGB_BLOCK: {
		Class:       CLASS_BC1_RGBA,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC2_UNORM_BLOCK: {
		Class:       CLASS_BC2,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC2_SRGB_BLOCK: {
		Class:       CLASS_BC2,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC3_UNORM_BLOCK: {
		Class:       CLASS_BC3,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC3_SRGB_BLOCK: {
		Class:       CLASS_BC3,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC4_UNORM_BLOCK: {
		Class:       CLASS_BC4,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC4_SNORM_BLOCK: {
		Class:       CLASS_BC4,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC5_UNORM_BLOCK: {
		Class:       CLASS_BC5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC5_SNORM_BLOCK: {
		Class:       CLASS_BC5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC6H_UFLOAT_BLOCK: {
		Class:       CLASS_BC6H,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC6H_SFLOAT_BLOCK: {
		Class:       CLASS_BC6H,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC7_UNORM_BLOCK: {
		Class:       CLASS_BC7,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_BC7_SRGB_BLOCK: {
		Class:       CLASS_BC7,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ETC2_R8G8B8_UNORM_BLOCK: {
		Class:       CLASS_ETC2_RGB,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ETC2_R8G8B8_SRGB_BLOCK: {
		Class:       CLASS_ETC2_RGB,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ETC2_R8G8B8A1_UNORM_BLOCK: {
		Class:       CLASS_ETC2_RGBA,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ETC2_R8G8B8A1_SRGB_BLOCK: {
		Class:       CLASS_ETC2_RGBA,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK: {
		Class:       CLASS_ETC2_EAC_RGBA,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ETC2_R8G8B8A8_SRGB_BLOCK: {
		Class:       CLASS_ETC2_EAC_RGBA,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_EAC_R11_UNORM_BLOCK: {
		Class:       CLASS_EAC_R,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_EAC_R11_SNORM_BLOCK: {
		Class:       CLASS_EAC_R,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_EAC_R11G11_UNORM_BLOCK: {
		Class:       CLASS_EAC_RG,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_EAC_R11G11_SNORM_BLOCK: {
		Class:       CLASS_EAC_RG,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_4x4_UNORM_BLOCK: {
		Class:       CLASS_ASTC_4X4,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_4x4_SRGB_BLOCK: {
		Class:       CLASS_ASTC_4X4,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_5x4_UNORM_BLOCK: {
		Class:       CLASS_ASTC_5X4,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 5, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_5x4_SRGB_BLOCK: {
		Class:       CLASS_ASTC_5X4,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 5, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_5x5_UNORM_BLOCK: {
		Class:       CLASS_ASTC_5X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 5, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_5x5_SRGB_BLOCK: {
		Class:       CLASS_ASTC_5X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 5, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_6x5_UNORM_BLOCK: {
		Class:       CLASS_ASTC_6X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 6, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_6x5_SRGB_BLOCK: {
		Class:       CLASS_ASTC_6X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 6, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_6x6_UNORM_BLOCK: {
		Class:       CLASS_ASTC_6X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 6, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_6x6_SRGB_BLOCK: {
		Class:       CLASS_ASTC_6X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 6, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_8x5_UNORM_BLOCK: {
		Class:       CLASS_ASTC_8X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_8x5_SRGB_BLOCK: {
		Class:       CLASS_ASTC_8X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_8x6_UNORM_BLOCK: {
		Class:       CLASS_ASTC_8X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_8x6_SRGB_BLOCK: {
		Class:       CLASS_ASTC_8X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_8x8_UNORM_BLOCK: {
		Class:       CLASS_ASTC_8X8,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 8, Depth: 1},
	},
	vk.FORMAT_ASTC_8x8_SRGB_BLOCK: {
		Class:       CLASS_ASTC_8X8,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 8, Depth: 1},
	},
	vk.FORMAT_ASTC_10x5_UNORM_BLOCK: {
		Class:       CLASS_ASTC_10X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_10x5_SRGB_BLOCK: {
		Class:       CLASS_ASTC_10X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_10x6_UNORM_BLOCK: {
		Class:       CLASS_ASTC_10X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_10x6_SRGB_BLOCK: {
		Class:       CLASS_ASTC_10X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_10x8_UNORM_BLOCK: {
		Class:       CLASS_ASTC_10X8,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 8, Depth: 1},
	},
	vk.FORMAT_ASTC_10x8_SRGB_BLOCK: {
		Class:       CLASS_ASTC_10X8,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 8, Depth: 1},
	},
	vk.FORMAT_ASTC_10x10_UNORM_BLOCK: {
		Class:       CLASS_ASTC_10X10,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 10, Depth: 1},
	},
	vk.FORMAT_ASTC_10x10_SRGB_BLOCK: {
		Class:       CLASS_ASTC_10X10,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 10, Depth: 1},
	},
	vk.FORMAT_ASTC_12x10_UNORM_BLOCK: {
		Class:       CLASS_ASTC_12X10,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 12, Height: 10, Depth: 1},
	},
	vk.FORMAT_ASTC_12x10_SRGB_BLOCK: {
		Class:       CLASS_ASTC_12X10,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 12, Height: 10, Depth: 1},
	},
	vk.FORMAT_ASTC_12x12_UNORM_BLOCK: {
		Class:       CLASS_ASTC_12X12,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 12, Height: 12, Depth: 1},
	},
	vk.FORMAT_ASTC_12x12_SRGB_BLOCK: {
		Class:       CLASS_ASTC_12X12,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 12, Height: 12, Depth: 1},
	},
	vk.FORMAT_G8B8G8R8_422_UNORM: {
		Class:       CLASS_32BIT_G8B8G8R8,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_B8G8R8G8_422_UNORM: {
		Class:       CLASS_32BIT_B8G8R8G8,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_G8_B8_R8_3PLANE_420_UNORM: {
		Class:       CLASS_8BIT_3PLANE_420,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G8_B8R8_2PLANE_420_UNORM: {
		Class:       CLASS_8BIT_2PLANE_420,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G8_B8_R8_3PLANE_422_UNORM: {
		Class:       CLASS_8BIT_3PLANE_422,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G8_B8R8_2PLANE_422_UNORM: {
		Class:       CLASS_8BIT_2PLANE_422,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G8_B8_R8_3PLANE_444_UNORM: {
		Class:       CLASS_8BIT_3PLANE_444,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R10X6_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R10X6G10X6_UNORM_2PACK16: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R10X6G10X6B10X6A10X6_UNORM_4PACK16: {
		Class:       CLASS_64BIT_R10G10B10A10,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6B10X6G10X6R10X6_422_UNORM_4PACK16: {
		Class:       CLASS_64BIT_G10B10G10R10,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_B10X6G10X6R10X6G10X6_422_UNORM_4PACK16: {
		Class:       CLASS_64BIT_B10G10R10G10,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16: {
		Class:       CLASS_10BIT_3PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6_B10X6R10X6_2PLANE_420_UNORM_3PACK16: {
		Class:       CLASS_10BIT_2PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16: {
		Class:       CLASS_10BIT_3PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6_B10X6R10X6_2PLANE_422_UNORM_3PACK16: {
		Class:       CLASS_10BIT_2PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16: {
		Class:       CLASS_10BIT_3PLANE_444,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R12X4_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R12X4G12X4_UNORM_2PACK16: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R12X4G12X4B12X4A12X4_UNORM_4PACK16: {
		Class:       CLASS_64BIT_R12G12B12A12,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4B12X4G12X4R12X4_422_UNORM_4PACK16: {
		Class:       CLASS_64BIT_G12B12G12R12,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_B12X4G12X4R12X4G12X4_422_UNORM_4PACK16: {
		Class:       CLASS_64BIT_B12G12R12G12,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16: {
		Class:       CLASS_12BIT_3PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4_B12X4R12X4_2PLANE_420_UNORM_3PACK16: {
		Class:       CLASS_12BIT_2PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16: {
		Class:       CLASS_12BIT_3PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4_B12X4R12X4_2PLANE_422_UNORM_3PACK16: {
		Class:       CLASS_12BIT_2PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16: {
		Class:       CLASS_12BIT_3PLANE_444,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16B16G16R16_422_UNORM: {
		Class:       CLASS_64BIT_G16B16G16R16,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_B16G16R16G16_422_UNORM: {
		Class:       CLASS_64BIT_B16G16R16G16,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 2, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16_B16_R16_3PLANE_420_UNORM: {
		Class:       CLASS_16BIT_3PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16_B16R16_2PLANE_420_UNORM: {
		Class:       CLASS_16BIT_2PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16_B16_R16_3PLANE_422_UNORM: {
		Class:       CLASS_16BIT_3PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16_B16R16_2PLANE_422_UNORM: {
		Class:       CLASS_16BIT_2PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16_B16_R16_3PLANE_444_UNORM: {
		Class:       CLASS_16BIT_3PLANE_444,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_PVRTC1_2BPP_UNORM_BLOCK_IMG: {
		Class:       CLASS_PVRTC1_2BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 8, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC1_4BPP_UNORM_BLOCK_IMG: {
		Class:       CLASS_PVRTC1_4BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC2_2BPP_UNORM_BLOCK_IMG: {
		Class:       CLASS_PVRTC2_2BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 8, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC2_4BPP_UNORM_BLOCK_IMG: {
		Class:       CLASS_PVRTC2_4BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC1_2BPP_SRGB_BLOCK_IMG: {
		Class:       CLASS_PVRTC1_2BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 8, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC1_4BPP_SRGB_BLOCK_IMG: {
		Class:       CLASS_PVRTC1_4BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC2_2BPP_SRGB_BLOCK_IMG: {
		Class:       CLASS_PVRTC2_2BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 8, Height: 4, Depth: 1},
	},
	vk.FORMAT_PVRTC2_4BPP_SRGB_BLOCK_IMG: {
		Class:       CLASS_PVRTC2_4BPP,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_4x4_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_4X4,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 4, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_5x4_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_5X4,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 5, Height: 4, Depth: 1},
	},
	vk.FORMAT_ASTC_5x5_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_5X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 5, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_6x5_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_6X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 6, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_6x6_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_6X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 6, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_8x5_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_8X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_8x6_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_8X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_8x8_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_8X8,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 8, Height: 8, Depth: 1},
	},
	vk.FORMAT_ASTC_10x5_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_10X5,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 5, Depth: 1},
	},
	vk.FORMAT_ASTC_10x6_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_10X6,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 6, Depth: 1},
	},
	vk.FORMAT_ASTC_10x8_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_10X8,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 8, Depth: 1},
	},
	vk.FORMAT_ASTC_10x10_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_10X10,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 10, Height: 10, Depth: 1},
	},
	vk.FORMAT_ASTC_12x10_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_12X10,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 12, Height: 10, Depth: 1},
	},
	vk.FORMAT_ASTC_12x12_SFLOAT_BLOCK: {
		Class:       CLASS_ASTC_12X12,
		BlockSize:   16,
		BlockExtent: vk.Extent3D{Width: 12, Height: 12, Depth: 1},
	},
	vk.FORMAT_G8_B8R8_2PLANE_444_UNORM: {
		Class:       CLASS_8BIT_2PLANE_444,
		BlockSize:   3,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G10X6_B10X6R10X6_2PLANE_444_UNORM_3PACK16: {
		Class:       CLASS_10BIT_2PLANE_444,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G12X4_B12X4R12X4_2PLANE_444_UNORM_3PACK16: {
		Class:       CLASS_12BIT_2PLANE_444,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G16_B16R16_2PLANE_444_UNORM: {
		Class:       CLASS_16BIT_2PLANE_444,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A4R4G4B4_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_A4B4G4R4_UNORM_PACK16: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R16G16_SFIXED5_NV: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R10X6_UINT_PACK16_ARM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R10X6G10X6_UINT_2PACK16_ARM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R10X6G10X6B10X6A10X6_UINT_4PACK16_ARM: {
		Class:       CLASS_64BIT_R10G10B10A10,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R12X4_UINT_PACK16_ARM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R12X4G12X4_UINT_2PACK16_ARM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R12X4G12X4B12X4A12X4_UINT_4PACK16_ARM: {
		Class:       CLASS_64BIT_R12G12B12A12,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R14X2_UINT_PACK16_ARM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R14X2G14X2_UINT_2PACK16_ARM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R14X2G14X2B14X2A14X2_UINT_4PACK16_ARM: {
		Class:       CLASS_64BIT_R14G14B14A14,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R14X2_UNORM_PACK16_ARM: {
		Class:       CLASS_16BIT,
		BlockSize:   2,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R14X2G14X2_UNORM_2PACK16_ARM: {
		Class:       CLASS_32BIT,
		BlockSize:   4,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R14X2G14X2B14X2A14X2_UNORM_4PACK16_ARM: {
		Class:       CLASS_64BIT_R14G14B14A14,
		BlockSize:   8,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G14X2_B14X2R14X2_2PLANE_420_UNORM_3PACK16_ARM: {
		Class:       CLASS_14BIT_2PLANE_420,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_G14X2_B14X2R14X2_2PLANE_422_UNORM_3PACK16_ARM: {
		Class:       CLASS_14BIT_2PLANE_422,
		BlockSize:   6,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
	vk.FORMAT_R8_BOOL_ARM: {
		Class:       CLASS_8BIT,
		BlockSize:   1,
		BlockExtent: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
	},
}
