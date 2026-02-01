package pathtracer

import "worldspawn/gpu"

type Film struct {
	Extent [2]int

	Color *gpu.Image

	DiffuseAlbedo      *gpu.Image
	NormalAndRoughness *gpu.Image
	Depth              *gpu.Image
	Motion             *gpu.Image
}

type _Film struct {
	Color gpu.ImageDescriptors

	DiffuseAlbedo      gpu.ImageDescriptors
	NormalAndRoughness gpu.ImageDescriptors
	Depth              gpu.ImageDescriptors
	Motion             gpu.ImageDescriptors
}
