package renderer

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: neural textures? We might need to keep this Texture stuff.

type Texture struct {
	Image *gpu.Image
}

func NewTexture(_type vk.ImageViewType, extent gpu.Int3, mipLevels, layers int, format gpu.Format) *Texture {
	texture := new(Texture)

	// TODO: should be sparse
	texture.Image = gpu.NewImage(&gpu.ImageConfig{
		Dim:       vkImageViewTypeToDimension(_type),
		Extent:    extent,
		Layers:    layers,
		MipLevels: mipLevels,
		Samples:   1,
		Format:    format,
		Usage:     gpu.ImageUsageSampling,
	})

	return texture
}

func vkImageViewTypeToDimension(viewType vk.ImageViewType) gpu.ImageDim {
	switch viewType {
	case vk.IMAGE_VIEW_TYPE_2D, vk.IMAGE_VIEW_TYPE_2D_ARRAY:
		return gpu.ImageDim2D
	case vk.IMAGE_VIEW_TYPE_CUBE, vk.IMAGE_VIEW_TYPE_CUBE_ARRAY:
		return gpu.ImageDimCube
	default:
		panic("unreachable")
	}
}
