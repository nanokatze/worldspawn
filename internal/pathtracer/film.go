package pathtracer

import "worldspawn/gpu"

type Film struct {
	Extent [2]int

	Color         *gpu.Image
	DiffuseAlbedo *gpu.Image
	Normal        *gpu.Image // TODO: fold in roughness here
}

type _Film struct {
	Color         gpu.ImageDescriptors
	DiffuseAlbedo gpu.ImageDescriptors
	Normal        gpu.ImageDescriptors
}
