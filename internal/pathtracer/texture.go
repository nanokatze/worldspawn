package pathtracer

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: neural textures? We might need to keep this Texture stuff, but creation
// and memory management should be on the user, just like with Mesh.

type Texture struct {
	Image *gpu.Image
}

func NewTexture(_type vk.ImageViewType, extent [3]int, mipLevels, layers int, format gpu.Format) *Texture {
	texture := new(Texture)

	texture.Image = gpu.NewImage(format, extent[:2], gpu.WithLayers{0, layers}, gpu.WithMips{0, mipLevels}, gpu.WithUsage(vk.IMAGE_USAGE_SAMPLED_BIT))

	if _type == vk.IMAGE_VIEW_TYPE_CUBE || _type == vk.IMAGE_VIEW_TYPE_CUBE_ARRAY {
		texture.Image = texture.Image.SubImage(gpu.ViewAs(gpu.ImageDimCube))
	}

	return texture
}
