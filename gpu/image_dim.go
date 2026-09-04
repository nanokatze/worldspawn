package gpu

import "worldspawn/gpu/vk"

const maxDimensions = 3

// Bits 0:6 indicate the number of dimensions, which is always at least 1.
// Bit 7 indicates the cube flag. Only valid for 2D images.
type imageDim uint8

func makeImageDim(dimensions int) imageDim {
	if !(1 <= dimensions && dimensions <= maxDimensions) {
		panic("bad number of dimensions")
	}
	return imageDim(uint8(dimensions))
}

func (dim imageDim) dimensions() int {
	return int(dim &^ 0x80)
}

func (dim imageDim) isCube() bool { return dim&0x80 != 0 }

func (dim imageDim) vkImageType() vk.ImageType {
	switch dim.dimensions() {
	case 1:
		return vk.IMAGE_TYPE_1D
	case 2:
		return vk.IMAGE_TYPE_2D
	case 3:
		return vk.IMAGE_TYPE_3D
	default:
		panic("unreachable")
	}
}

func (dim imageDim) vkImageViewType() vk.ImageViewType {
	switch dim {
	case 1:
		return vk.IMAGE_VIEW_TYPE_1D_ARRAY
	case 2:
		return vk.IMAGE_VIEW_TYPE_2D_ARRAY
	case 2 | 0x80:
		return vk.IMAGE_VIEW_TYPE_CUBE_ARRAY
	case 3:
		return vk.IMAGE_VIEW_TYPE_3D
	default:
		panic("unreachable")
	}
}
